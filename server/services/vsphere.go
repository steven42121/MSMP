// Package services 提供 vSphere/ESXi 虚拟化管理能力。
package services

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"MSMP/server/db"
	"MSMP/server/models"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// VSphereManager 封装 vSphere 连接与虚拟化管理操作。
type VSphereManager struct {
	client  *govmomi.Client
	binding *models.ChannelBinding
}

// VMInfo 虚拟机信息。
type VMInfo struct {
	Name          string `json:"name"`
	PowerState    string `json:"power_state"`
	NumCPU        int    `json:"num_cpu"`
	MemoryMB      int    `json:"memory_mb"`
	IPAddress     string `json:"ip_address"`
	GuestOS       string `json:"guest_os"`
	IsTemplate    bool   `json:"is_template"`
	SnapshotCount int    `json:"snapshot_count"`
}

// DatastoreInfo 数据存储信息。
type DatastoreInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Capacity  int64  `json:"capacity"`
	FreeSpace int64  `json:"free_space"`
	Accessible bool `json:"accessible"`
}

// parseVSphereAddress 规范化地址为 SDK URL。
func parseVSphereAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	return "https://" + addr + "/sdk"
}

// parseVSphereCredential 解析 username:password。
func parseVSphereCredential(secret string) (string, string) {
	parts := strings.SplitN(secret, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return secret, ""
}

// ConnectVSphere 建立到主机 vSphere 渠道的连接并返回管理器。
func ConnectVSphere(ctx context.Context, tenantID, hostID uint) (*VSphereManager, error) {
	if GlobalCredSvc == nil {
		return nil, fmt.Errorf("凭证服务未初始化")
	}

	var binding models.ChannelBinding
	if err := db.DB.Where("host_id = ? AND tenant_id = ? AND type = ? AND enabled = ?",
		hostID, tenantID, "vsphere", true).First(&binding).Error; err != nil {
		return nil, fmt.Errorf("主机未配置 vSphere 渠道")
	}

	secret, err := GlobalCredSvc.Decrypt(binding.Credential)
	if err != nil {
		return nil, fmt.Errorf("凭证解密失败: %v", err)
	}

	username, password := parseVSphereCredential(secret)
	endpoint := parseVSphereAddress(binding.Address)

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("无效的 vSphere 地址: %v", err)
	}

	c, err := govmomi.NewClient(ctx, u, true) // 跳过证书校验（实验环境）
	if err != nil {
		return nil, fmt.Errorf("连接 vSphere 失败: %v", err)
	}

	if err := c.Login(ctx, url.UserPassword(username, password)); err != nil {
		return nil, fmt.Errorf("vSphere 认证失败: %v", err)
	}

	return &VSphereManager{client: c, binding: &binding}, nil
}

// Close 断开连接。
func (m *VSphereManager) Close(ctx context.Context) {
	if m.client != nil {
		_ = m.client.Logout(ctx)
	}
}

// ListVMs 列出所有虚拟机（遍历所有 datacenter）。
func (m *VSphereManager) ListVMs(ctx context.Context) ([]VMInfo, error) {
	finder := find.NewFinder(m.client.Client)

	// "..." 通配符递归遍历所有 datacenter 下的 VM
	vms, err := finder.VirtualMachineList(ctx, "...")
	if err != nil {
		return nil, fmt.Errorf("列出虚拟机失败: %v", err)
	}

	result := make([]VMInfo, 0, len(vms))
	for _, vm := range vms {
		var info mo.VirtualMachine
		if err := vm.Properties(ctx, vm.Reference(),
			[]string{"config", "summary", "snapshot"}, &info); err != nil {
			continue
		}

		vi := VMInfo{
			PowerState: string(info.Summary.Runtime.PowerState),
			IPAddress:  info.Summary.Guest.IpAddress,
		}
		if info.Config != nil {
			vi.Name = info.Config.Name
			vi.GuestOS = info.Config.GuestFullName
			vi.IsTemplate = info.Config.Template
			if info.Config.Hardware.NumCPU > 0 {
				vi.NumCPU = int(info.Config.Hardware.NumCPU)
			}
			vi.MemoryMB = int(info.Config.Hardware.MemoryMB)
		}
		if vi.Name == "" {
			vi.Name = info.Summary.Config.Name
			vi.NumCPU = int(info.Summary.Config.NumCpu)
			vi.MemoryMB = int(info.Summary.Config.MemorySizeMB)
		}
		if info.Snapshot != nil {
			vi.SnapshotCount = countSnapshots(info.Snapshot.RootSnapshotList)
		}
		result = append(result, vi)
	}

	return result, nil
}

// countSnapshots 递归计数快照树。
func countSnapshots(list []types.VirtualMachineSnapshotTree) int {
	n := 0
	for _, s := range list {
		n++
		n += countSnapshots(s.ChildSnapshotList)
	}
	return n
}

// PowerVM 执行虚拟机电源操作。action: on/off/reset/suspend
func (m *VSphereManager) PowerVM(ctx context.Context, vmName, action string) error {
	vm, err := m.findVM(ctx, vmName)
	if err != nil {
		return err
	}

	var task *object.Task
	switch action {
	case "on":
		task, err = vm.PowerOn(ctx)
	case "off":
		task, err = vm.PowerOff(ctx)
	case "reset":
		task, err = vm.Reset(ctx)
	case "suspend":
		task, err = vm.Suspend(ctx)
	default:
		return fmt.Errorf("无效的电源操作: %s", action)
	}
	if err != nil {
		return fmt.Errorf("操作失败: %v", err)
	}
	if task != nil {
		if err := task.Wait(ctx); err != nil {
			return fmt.Errorf("操作未完成: %v", err)
		}
	}
	return nil
}

// findVM 按名称查找虚拟机。
func (m *VSphereManager) findVM(ctx context.Context, name string) (*object.VirtualMachine, error) {
	finder := find.NewFinder(m.client.Client)
	vm, err := finder.VirtualMachine(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("找不到虚拟机 %s: %v", name, err)
	}
	return vm, nil
}

// ListDatastores 列出所有数据存储。
func (m *VSphereManager) ListDatastores(ctx context.Context) ([]DatastoreInfo, error) {
	finder := find.NewFinder(m.client.Client)

	dss, err := finder.DatastoreList(ctx, "...")
	if err != nil {
		return nil, fmt.Errorf("列出数据存储失败: %v", err)
	}

	result := make([]DatastoreInfo, 0, len(dss))
	for _, ds := range dss {
		var info mo.Datastore
		if err := ds.Properties(ctx, ds.Reference(), []string{"summary"}, &info); err != nil {
			continue
		}
		result = append(result, DatastoreInfo{
			Name:       info.Summary.Name,
			Type:       info.Summary.Type,
			Capacity:   info.Summary.Capacity,
			FreeSpace:  info.Summary.FreeSpace,
			Accessible: info.Summary.Accessible,
		})
	}
	return result, nil
}