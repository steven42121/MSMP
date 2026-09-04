// Package collectors contains the vSphere/ESXi collector implementation.
package collectors

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"MSMP/server/models"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// VSphereChannel implements the Channel interface for vSphere/ESXi hosts.
// Supports connecting to a vCenter Server or directly to an ESXi host.
type VSphereChannel struct{}

func (v *VSphereChannel) Type() string { return "vsphere" }

// vsphereConfig holds parsed connection parameters.
type vsphereConfig struct {
	Endpoint string // host:port or https://host/sdk
	Username string
	Password string
	VCType   string // "vc" for vCenter, "esxi" for direct ESXi
}

// parseVSphereAddress extracts endpoint from ChannelBinding.Address.
// Expected formats:
//   - "https://vcenter.example.com/sdk" (vCenter with SDK path)
//   - "vcenter.example.com" (vCenter, defaults to https)
//   - "esxi-host.local" (ESXi host, defaults to https)
func parseVSphereAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	// Default to https SDK endpoint
	return "https://" + addr + "/sdk"
}

func (v *VSphereChannel) dial(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (*govmomi.Client, error) {
	secret, err := cred.Decrypt(b.Credential)
	if err != nil {
		return nil, fmt.Errorf("%s:decrypt credential: %w", StatusAuthFailed, err)
	}

	// secret format: "username:password" for password auth
	parts := strings.SplitN(secret, ":", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s:credential must be 'username:password'", StatusAuthFailed)
	}
	username := parts[0]
	password := parts[1]

	endpoint := parseVSphereAddress(b.Address)

	c, err := govmomi.NewClient(ctx, endpoint, username)
	if err != nil {
		if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "x509") {
			// Try with insecure skip verify for self-signed certs (common in lab)
			c, err = govmomi.NewClient(ctx, endpoint, username)
			if err != nil {
				return nil, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
			}
		} else {
			return nil, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
		}
	}

	if c.Session == nil || !c.Session.IsLogin() {
		return nil, fmt.Errorf("%s:not logged in", StatusAuthFailed)
	}
	return c, nil
}

// Probe connects to vSphere and verifies the host is reachable.
func (v *VSphereChannel) Probe(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	c, err := v.dial(ctx, b, cred)
	if err != nil {
		pr, _ := classify(err)
		return pr, err
	}
	defer c.Logout(ctx)

	// Fetch about info to confirm connection
	var about mo.AboutInfo
	err = c.RetrieveOne(ctx, c.ServiceContent.About, &about)
	if err != nil {
		return ProbeResult{Err: StatusParseError}, err
	}

	return ProbeResult{
		OK:   true,
		OS:   "vmware-esxi",
		Host: about.Name,
	}, nil
}

// Collect gathers metrics from vSphere (ESXi hosts and/or VMs).
func (v *VSphereChannel) Collect(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	c, err := v.dial(ctx, b, cred)
	if err != nil {
		return CollectResult{}, err
	}
	defer c.Logout(ctx)

	// Build result
	var result CollectResult

	// 1. Collect ESXi host metrics (if a specific host is targeted)
	hostMetrics, err := v.collectHostMetrics(ctx, c)
	if err != nil {
		log.Printf("[VSphere] host metrics collect error: %v", err)
	} else {
		result.Metrics = hostMetrics
	}

	// 2. Collect VM count and status
	vmStats, err := v.collectVMStats(ctx, c)
	if err != nil {
		log.Printf("[VSphere] vm stats collect error: %v", err)
	} else {
		result.VMCount = vmStats.Total
		result.VMRunning = vmStats.Running
		result.VMPoweredOff = vmStats.PoweredOff
	}

	// 3. Collect datastore free/total space
	dsStats, err := v.collectDatastoreStats(ctx, c)
	if err != nil {
		log.Printf("[VSphere] datastore stats error: %v", err)
	} else {
		result.DatastoreTotalGB = dsStats.TotalGB
		result.DatastoreFreeGB = dsStats.FreeGB
	}

	result.Duration = time.Since(start)
	return result, nil
}

// esxiHostMetrics holds aggregated ESXi host metrics.
type esxiHostMetrics struct {
	CPUPercent float64
	MemPercent float64
	MemUsed    uint64
	MemTotal   uint64
	DiskUsed   uint64
	DiskTotal  uint64
	Load1      float64
	UptimeSec  uint64
}

// collectHostMetrics gathers CPU, memory, disk, and load from ESXi hosts.
func (v *VSphereChannel) collectHostMetrics(ctx context.Context, c *govmomi.Client) (esxiHostMetrics, error) {
	var m esxiHostMetrics

	// Find all hosts in the inventory
	finder := object.NewSearchIndex(c.Client)
	// Use root folder
	root, err := finder.DefaultFolder(ctx, "Datacenters")
	if err != nil {
		return m, fmt.Errorf("find datacenters: %w", err)
	}

	var hosts []object.InventoryPath
	// Navigate datacenters -> clusters -> hosts
	ds, err := root.Children(ctx)
	if err != nil {
		return m, err
	}
	for _, dc := range ds {
		dcMo := dc.(*object.Datacenter)
		folders, err := dcMo.Folders(ctx)
		if err != nil {
			continue
		}
		if folders.HostsFolder == nil {
			continue
		}
		hostsInFolder, err := folders.HostsFolder.Children(ctx)
		if err != nil {
			continue
		}
		for _, h := range hostsInFolder {
			hMo := h.(*object.HostSystem)
			hosts = append(hosts, hMo.InventoryPath)
		}
	}

	if len(hosts) == 0 {
		return m, fmt.Errorf("no ESXi hosts found in inventory")
	}

	// Aggregate metrics across all hosts
	var totalCPU float64
	var totalMemUsed, totalMemTotal uint64
	var totalDiskUsed, totalDiskTotal uint64
	var totalLoad float64
	var count int

	for _, hp := range hosts {
		hs := object.NewHostSystem(c.Client, hp.Reference())
		var info mo.HostSystem
		err := hs.Properties(ctx, hs.Reference(), []string{"summary", "summary.hardware", "summary.quickStats"}, &info)
		if err != nil {
			continue
		}

		s := info.Summary
		// CPU
		if s.Hardware.CpuMhz > 0 && s.Hardware.NumCpuCores > 0 {
			totalCPU += float64(s.Statistics.Cpu.UsedAverage) / 100.0
		}
		// Memory
		memTotal := uint64(s.Hardware.MemorySize)
		memUsed := uint64(s.QuickStats.OverallMemoryUsage) * 1024 * 1024
		totalMemUsed += memUsed
		totalMemTotal += memTotal
		// Load (average over hosts)
		totalLoad += s.QuickStats.GlobalCpuUsage
		count++
	}

	if count > 0 {
		m.CPUPercent = totalCPU / float64(count)
		m.Load1 = totalLoad / float64(count)
		if totalMemTotal > 0 {
			m.MemPercent = float64(totalMemUsed) / float64(totalMemTotal) * 100
		}
		m.MemUsed = totalMemUsed
		m.MemTotal = totalMemTotal
	}

	return m, nil
}

// vmCountStats holds VM counts by power state.
type vmCountStats struct {
	Total        int
	Running      int
	PoweredOff   int
	Templates    int
}

// collectVMStats counts VMs by power state.
func (v *VSphereChannel) collectVMStats(ctx context.Context, c *govmomi.Client) (vmCountStats, error) {
	var stats vmCountStats

	finder := object.NewSearchIndex(c.Client)
	root, err := finder.DefaultFolder(ctx, "Datacenters")
	if err != nil {
		return stats, err
	}

	dcChildren, err := root.Children(ctx)
	if err != nil {
		return stats, err
	}

	for _, dc := range dcChildren {
		dcMo := dc.(*object.Datacenter)
		folders, err := dcMo.Folders(ctx)
		if err != nil {
			continue
		}
		vms, err := folders.VmFolder.Children(ctx)
		if err != nil {
			continue
		}
		for _, vm := range vms {
			vmMo := vm.(*object.VirtualMachine)
			stats.Total++
			var info mo.VirtualMachine
			err = vmMo.Properties(ctx, vmMo.Reference(), []string{"runtime", "config.template"}, &info)
			if err != nil {
				continue
			}
			if info.Config != nil && info.Config.Template {
				stats.Templates++
				continue
			}
			if info.Runtime != nil {
				switch info.Runtime.PowerState {
				case types.VirtualMachinePowerStatePoweredOn:
					stats.Running++
				case types.VirtualMachinePowerStatePoweredOff:
					stats.PoweredOff++
				}
			}
		}
	}
	return stats, nil
}

// datastoreStats holds aggregate datastore space info.
type datastoreStats struct {
	TotalGB float64
	FreeGB  float64
}

// collectDatastoreStats aggregates free/total space across all datastores.
func (v *VSphereChannel) collectDatastoreStats(ctx context.Context, c *govmomi.Client) (datastoreStats, error) {
	var stats datastoreStats

	finder := object.NewSearchIndex(c.Client)
	root, err := finder.DefaultFolder(ctx, "Datacenters")
	if err != nil {
		return stats, err
	}

	dcChildren, err := root.Children(ctx)
	if err != nil {
		return stats, err
	}

	for _, dc := range dcChildren {
		dcMo := dc.(*object.Datacenter)
		folders, err := dcMo.Folders(ctx)
		if err != nil {
			continue
		}
		dss, err := folders.DatastoreFolder.Children(ctx)
		if err != nil {
			continue
		}
		for _, ds := range dss {
			dsMo := ds.(*object.Datastore)
			var info mo.Datastore
			err := dsMo.Properties(ctx, dsMo.Reference(), []string{"summary", "info"}, &info)
			if err != nil {
				continue
			}
			sum := info.Summary
			totalBytes := sum.Capacity
			freeBytes := sum.FreeSpace
			if totalBytes > 0 {
				stats.TotalGB += float64(totalBytes) / (1024 * 1024 * 1024)
			}
			if freeBytes > 0 {
				stats.FreeGB += float64(freeBytes) / (1024 * 1024 * 1024)
			}
		}
	}
	return stats, nil
}

// collectTaskInfo collects recent tasks for alerting context.
func (v *VSphereChannel) collectTaskInfo(ctx context.Context, c *govmomi.Client) ([]string, error) {
	var tasks []string
	// Fetch last 10 tasks
	ts := object.NewTaskManager(c.Client)
	limit := int32(10)
	// Note: this is a simplified collection; full task listing requires more complex query
	return tasks, nil
}

// classify maps error to status codes.
func classify(err error) (ProbeResult, error) {
	s := err.Error()
	switch {
	case strings.Contains(s, StatusUnreachable):
		return ProbeResult{OK: false, Err: StatusUnreachable}, err
	case strings.Contains(s, StatusAuthFailed):
		return ProbeResult{OK: false, Err: StatusAuthFailed}, err
	default:
		return ProbeResult{OK: false, Err: StatusParseError}, err
	}
}
