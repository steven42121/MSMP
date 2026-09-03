//go:build windows

package win

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"

	"MSMP/agent/common"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	getTickCount64 = kernel32.NewProc("GetTickCount64")
)

func getUptime() uint64 {
	ret, _, _ := getTickCount64.Call()
	return uint64(ret) / 1000
}

func CollectAssetInfoFull() common.AgentInfo {
	hostname, _ := os.Hostname()
	hinfo, _ := host.Info()
	vmem, _ := mem.VirtualMemory()
	diskParts, _ := disk.Partitions(false)

	var totalDisk uint64
	for _, d := range diskParts {
		if usage, err := disk.Usage(d.Mountpoint); err == nil {
			totalDisk += usage.Total
		}
	}

	var cpuModel string
	cpuCores := runtime.NumCPU()
	if ci, err := cpu.Info(); err == nil && len(ci) > 0 {
		cpuModel = ci[0].ModelName
		cpuCores = len(ci)
	}

	ip := common.GetLocalIP()
	publicIP := common.GetPublicIP()

	return common.AgentInfo{
		UUID:              common.GetEnv("AGENT_UUID", hostname),
		Hostname:          hostname,
		OS:                "windows",
		OSVersion:         fmt.Sprintf("%s %s", hinfo.Platform, hinfo.PlatformVersion),
		Arch:              runtime.GOARCH,
		IP:                ip,
		PublicIP:          publicIP,
		CPUModel:          cpuModel,
		CPUCores:          cpuCores,
		MemoryTotal:       vmem.Total,
		DiskTotal:         totalDisk,
		AgentVersion:      common.AgentVersion,
		DiskPartitions:    collectDiskPartitions(),
		NetworkInterfaces: collectNetInterfaces(),
		Processes:         collectProcesses(),
		GPUs:              common.CollectGPUs(),
		Temperatures:      collectTemperatures(),
	}
}

func collectDiskPartitions() []common.DiskPartition {
	parts, _ := disk.Partitions(false)
	var result []common.DiskPartition
	for _, p := range parts {
		dp := common.DiskPartition{
			Device:     p.Device,
			Mountpoint:  p.Mountpoint,
			Fstype:      p.Fstype,
		}
		if usage, err := disk.Usage(p.Mountpoint); err == nil {
			dp.Total = usage.Total
			dp.Used = usage.Used
			dp.UsedPercent = usage.UsedPercent
		}
		result = append(result, dp)
	}
	return result
}

func collectNetInterfaces() []common.NetInterface {
	addrs, _ := net.Interfaces()
	var result []common.NetInterface
	for _, a := range addrs {
		ni := common.NetInterface{
			Name: a.Name,
			Mac:  a.HardwareAddr,
		}
		for _, addr := range a.Addrs {
			ni.IP = addr.Addr
		}
		if counters, err := net.IOCounters(true); err == nil {
			for _, c := range counters {
				if c.Name == a.Name {
					ni.BytesRecv = c.BytesRecv
					ni.BytesSent = c.BytesSent
				}
			}
		}
		result = append(result, ni)
	}
	return result
}

func collectTemperatures() []common.TemperatureInfo {
	sensors, err := host.SensorsTemperatures()
	if err != nil {
		return nil
	}
	var result []common.TemperatureInfo
	for _, s := range sensors {
		result = append(result, common.TemperatureInfo{
			SensorKey: s.SensorKey,
			Temp:      s.Temperature,
			High:      s.High,
			Critical:  s.Critical,
		})
	}
	return result
}

func collectProcesses() []common.ProcessInfo {
	ps, err := process.Processes()
	if err != nil {
		return nil
	}
	var result []common.ProcessInfo
	for _, p := range ps {
		name, _ := p.Name()
		username, _ := p.Username()
		cpuPct, _ := p.CPUPercent()
		memPct, _ := p.MemoryPercent()
		result = append(result, common.ProcessInfo{
			PID:        p.Pid,
			Name:       name,
			Username:   username,
			CPUPercent: cpuPct,
			MemPercent: float64(memPct),
		})
	}
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}

func CollectMetrics() common.MetricData {
	vmem, _ := mem.VirtualMemory()
	diskParts, _ := disk.Partitions(false)

	var diskUsed, diskTotal uint64
	for _, d := range diskParts {
		if usage, err := disk.Usage(d.Mountpoint); err == nil {
			diskUsed += usage.Used
			diskTotal += usage.Total
		}
	}

	var cpuPercent float64
	if cp, err := cpu.Percent(time.Second, false); err == nil && len(cp) > 0 {
		cpuPercent = cp[0]
	}

	var loadAvg1, loadAvg5, loadAvg15 float64
	if lavg, err := load.Avg(); err == nil {
		loadAvg1 = lavg.Load1
		loadAvg5 = lavg.Load5
		loadAvg15 = lavg.Load15
	}

	var netRx, netTx uint64
	if netIO, err := net.IOCounters(false); err == nil && len(netIO) > 0 {
		netRx = netIO[0].BytesRecv
		netTx = netIO[0].BytesSent
	}

	return common.MetricData{
		CPUPercent: cpuPercent,
		MemPercent: vmem.UsedPercent,
		MemUsed:    vmem.Used,
		MemTotal:   vmem.Total,
		DiskUsed:   diskUsed,
		DiskTotal:  diskTotal,
		NetRxBps:   netRx,
		NetTxBps:   netTx,
		Load1:      loadAvg1,
		Load5:      loadAvg5,
		Load15:     loadAvg15,
		UptimeSec:  getUptime(),
	}
}