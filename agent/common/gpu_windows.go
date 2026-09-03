package common

import (
	"os/exec"
	"strconv"
	"strings"
)

// CollectGPUsWindows 通过 WMIC 采集 Windows 上的 GPU 信息。
func CollectGPUsWindows() []GPUInfo {
	return collectGPUsViaWMIC()
}

func collectGPUsViaWMIC() []GPUInfo {
	out, err := exec.Command("wmic", "path", "win32_VideoController", "get",
		"Name,AdapterRAM,DriverVersion,PNPDeviceID", "/format:csv").Output()
	if err != nil {
		return nil
	}

	var gpus []GPUInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return nil
	}

	// 跳过标题行
	for _, line := range lines[1:] {
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}

		name := strings.TrimSpace(parts[1])
		if name == "" || strings.EqualFold(name, "name") {
			continue
		}

		adapterRAM, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
		driverVersion := strings.TrimSpace(parts[3])

		// 从 PNPDeviceID 提取厂商信息
		vendor := detectVendorFromPNP(parts[1]) // parts[1] 实际是 Name，PNPDeviceID 需要进一步解析

		gpu := GPUInfo{
			Name:          name,
			Vendor:        vendor,
			MemoryTotal:   adapterRAM,
			DriverVersion: driverVersion,
		}

		gpus = append(gpus, gpu)
	}

	return gpus
}

func detectVendorFromPNP(pnpID string) string {
	pnpID = strings.ToUpper(pnpID)
	switch {
	case strings.Contains(pnpID, "NVDA"):
		return "NVIDIA"
	case strings.Contains(pnpID, "AMD"):
		return "AMD"
	case strings.Contains(pnpID, "INTEL"):
		return "Intel"
	default:
		return "Unknown"
	}
}
