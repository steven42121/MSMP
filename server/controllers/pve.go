// Package controllers 提供 Proxmox VE 管理 API。
package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"MSMP/server/collectors"
	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/services"
)

// requirePVEAdmin 校验 admin 角色 + pve 渠道存在。
func requirePVEAdmin(w http.ResponseWriter, r *http.Request, hostID, tenantID uint) bool {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可执行虚拟化操作"})
		return false
	}
	var count int64
	db.DB.Model(&models.ChannelBinding{}).
		Where("host_id = ? AND tenant_id = ? AND type = ? AND enabled = ?", hostID, tenantID, "pve", true).
		Count(&count)
	if count == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "主机未配置 Proxmox VE 渠道"})
		return false
	}
	return true
}

// connectPVE 解密渠道凭据并建立 PVE 客户端。
func connectPVE(r *http.Request, tenantID, hostID uint) (*services.PVEClient, *models.ChannelBinding, error) {
	if services.GlobalCredSvc == nil {
		return nil, nil, httpError("凭证服务未初始化")
	}

	var binding models.ChannelBinding
	if err := db.DB.Where("host_id = ? AND tenant_id = ? AND type = ? AND enabled = ?",
		hostID, tenantID, "pve", true).First(&binding).Error; err != nil {
		return nil, nil, httpError("主机未配置 Proxmox VE 渠道")
	}

	secret, err := services.GlobalCredSvc.Decrypt(binding.Credential)
	if err != nil {
		return nil, nil, httpError("凭证解密失败: " + err.Error())
	}

	username, password := collectors.ParsePVECredential(secret)
	client, err := services.NewPVEClient(r.Context(), binding.Address, username, password)
	if err != nil {
		return nil, nil, err
	}
	return client, &binding, nil
}

type httpErrorMessage string

func (e httpErrorMessage) Error() string { return string(e) }

func httpError(msg string) error { return httpErrorMessage(msg) }

// pveClient 校验权限并建立 PVE 连接，失败时已写响应并返回 false。
func pveClient(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID uint) (*services.PVEClient, bool) {
	if !requirePVEAdmin(w, r, host.ID, tenantID) {
		return nil, false
	}
	client, _, err := connectPVE(r, tenantID, host.ID)
	if err != nil {
		writeUpstreamErr(w, err)
		return nil, false
	}
	return client, true
}

// PVEGuestsHandler GET /api/hosts/{uuid}/pve/guests
func PVEGuestsHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {

	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	guests, err := client.ListGuests(r.Context())
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}

	auditLog(tenantID, userID, "pve_list_guests", "host:"+host.Hostname, 200)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"guests": guests,
		"count":  len(guests),
	})
}

// PVEGuestPowerHandler POST /api/hosts/{uuid}/pve/guests/power
// body: {"node": "pve1", "vmid": 100, "vmtype": "qemu"|"lxc", "action": "start"|"stop"|...}
func PVEGuestPowerHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {

	var req struct {
		Node   string `json:"node"`
		VMID   int    `json:"vmid"`
		GuestType string `json:"vmtype"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求体解析失败"})
		return
	}
	if req.Node == "" || req.VMID <= 0 || req.GuestType == "" || req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node, vmid, vmtype, action 均必填"})
		return
	}

	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	if err := client.PowerGuest(r.Context(), req.Node, req.GuestType, req.VMID, req.Action); err != nil {
		writeUpstreamErr(w, err)
		return
	}

	auditLog(tenantID, userID, "pve_power_guest", "guest:"+req.GuestType+":"+req.Node+"/"+strconv.Itoa(req.VMID)+":"+req.Action, 200)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node":   req.Node,
		"vmid":   req.VMID,
		"type":   req.GuestType,
		"action": req.Action,
		"done":   true,
	})
}

// PVEStorageHandler GET /api/hosts/{uuid}/pve/storage
func PVEStorageHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {

	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	storages, err := client.ListStorage(r.Context())
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"storages": storages,
		"count":    len(storages),
	})
}

// pveSnapshotReq 快照操作的公共请求体（node/vmtype/vmid + 名称/描述）。
type pveSnapshotReq struct {
	Node      string `json:"node"`
	GuestType string `json:"vmtype"`
	VMID      int    `json:"vmid"`
	Name      string `json:"name"`
	SnapName  string `json:"snap_name"`
	Desc      string `json:"description"`
}

func parsePVESnapshotReq(r *http.Request) (*pveSnapshotReq, string, bool) {
	var req pveSnapshotReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, "请求体解析失败", false
	}
	if req.Node == "" || req.VMID <= 0 || (req.GuestType != "qemu" && req.GuestType != "lxc") {
		return nil, "node, vmid, vmtype(qemu/lxc) 均必填", false
	}
	return &req, "", true
}

// PVESnapshotsHandler GET /api/hosts/{uuid}/pve/snapshots?node=&vmtype=&vmid=
func PVESnapshotsHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	node := r.URL.Query().Get("node")
	guestType := r.URL.Query().Get("vmtype")
	vmid, _ := strconv.Atoi(r.URL.Query().Get("vmid"))
	if node == "" || vmid <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node 和 vmid 必填"})
		return
	}

	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	snaps, err := client.ListSnapshots(r.Context(), node, guestType, vmid)
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"snapshots": snaps,
		"count":     len(snaps),
	})
}

// PVESnapshotActionHandler POST/DELETE /api/hosts/{uuid}/pve/snapshots
// POST 创建（body: {node, vmtype, vmid, name, description}）
// DELETE 删除（query: node/vmtype/vmid/snap_name）
func PVESnapshotActionHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {

	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req struct {
			Node      string `json:"node"`
			GuestType string `json:"vmtype"`
			VMID      int    `json:"vmid"`
			Name      string `json:"name"`
			Desc      string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求体解析失败"})
			return
		}
		if req.Node == "" || req.VMID <= 0 || (req.GuestType != "qemu" && req.GuestType != "lxc") || req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node, vmid, vmtype, name 均必填"})
			return
		}
		if err := client.CreateSnapshot(r.Context(), req.Node, req.GuestType, req.VMID, req.Name, req.Desc); err != nil {
			writeUpstreamErr(w, err)
			return
		}
		auditLog(tenantID, userID, "pve_create_snapshot", "snapshot:"+req.Node+"/"+strconv.Itoa(req.VMID)+":"+req.Name, 200)
		writeJSON(w, http.StatusOK, map[string]interface{}{"created": true, "name": req.Name})

	case http.MethodDelete:
		node := r.URL.Query().Get("node")
		guestType := r.URL.Query().Get("vmtype")
		vmid, _ := strconv.Atoi(r.URL.Query().Get("vmid"))
		snapName := r.URL.Query().Get("snap_name")
		if node == "" || vmid <= 0 || snapName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node, vmid, snap_name 必填"})
			return
		}
		if err := client.DeleteSnapshot(r.Context(), node, guestType, vmid, snapName); err != nil {
			writeUpstreamErr(w, err)
			return
		}
		auditLog(tenantID, userID, "pve_delete_snapshot", "snapshot:"+node+"/"+strconv.Itoa(vmid)+":"+snapName, 200)
		writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "name": snapName})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// PVEClusterHandler GET /api/hosts/{uuid}/pve/cluster
// 返回集群聚合资源（节点/VM/容器/存储）。
func PVEClusterHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	resources, err := client.ClusterResources(r.Context())
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"resources": resources,
		"count":     len(resources),
	})
}

// PVENetworksHandler GET /api/hosts/{uuid}/pve/networks
// 遍历所有在线节点的网络配置。
func PVENetworksHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	nodes, err := client.Nodes(r.Context())
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}

	type netEntry struct {
		Node string `json:"node"`
		services.PVENetwork
	}
	all := make([]netEntry, 0)
	for _, node := range nodes {
		if node.Status != "online" {
			continue
		}
		nets, err := client.ListNetworks(r.Context(), node.Node)
		if err != nil {
			continue
		}
		for _, n := range nets {
			all = append(all, netEntry{Node: node.Node, PVENetwork: n})
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"networks": all,
		"count":    len(all),
	})
}

// PVEGuestDetailHandler GET /api/hosts/{uuid}/pve/guest?node=&vmtype=&vmid=
// 返回配置 + 实时资源，供详情弹窗展示（QEMU/LXC 通用）。
func PVEGuestDetailHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	node := r.URL.Query().Get("node")
	guestType := r.URL.Query().Get("vmtype")
	vmid, _ := strconv.Atoi(r.URL.Query().Get("vmid"))
	if node == "" || vmid <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node 和 vmid 必填"})
		return
	}

	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	cfg, err := client.GetGuestConfig(r.Context(), node, guestType, vmid)
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}
	runtime, _ := client.GetGuestRuntime(r.Context(), node, guestType, vmid)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node":      node,
		"vmtype":    guestType,
		"vmid":      vmid,
		"config":    cfg,
		"runtime":   runtime,
	})
}

// PVEBackupsHandler GET /api/hosts/{uuid}/pve/backups
func PVEBackupsHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	jobs, err := client.ListBackupJobs(r.Context())
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// PVEBackupContentHandler GET /api/hosts/{uuid}/pve/backups/content?node=&storage=
func PVEBackupContentHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	node := r.URL.Query().Get("node")
	storage := r.URL.Query().Get("storage")
	if node == "" || storage == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node 和 storage 必填"})
		return
	}

	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}

	files, err := client.ListBackupContent(r.Context(), node, storage)
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"files": files,
		"count": len(files),
	})
}

// PVESnapshotRollbackHandler POST /api/hosts/{uuid}/pve/snapshots/rollback
func PVESnapshotRollbackHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	req, msg, ok := parsePVESnapshotReq(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if req.SnapName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "snap_name 必填"})
		return
	}

	client, ok := pveClient(w, r, host, tenantID)
	if !ok {
		return
	}
	if err := client.RollbackSnapshot(r.Context(), req.Node, req.GuestType, req.VMID, req.SnapName); err != nil {
		writeUpstreamErr(w, err)
		return
	}

	auditLog(tenantID, userID, "pve_rollback_snapshot", "snapshot:"+req.Node+"/"+strconv.Itoa(req.VMID)+":"+req.SnapName, 200)
	writeJSON(w, http.StatusOK, map[string]interface{}{"rolled_back": true, "name": req.SnapName})
}
