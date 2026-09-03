package common

import (
	"os/exec"
	"strconv"
	"strings"
)

// CollectGPUs 通过 nvidia-smi 采集 NVIDIA GPU 信息。
// 优先使用二进制名称 "nvidia-smi"，其次尝试其绝对路径。
// 查询列顺序必须与 parseGPU 的字段顺序保持一致。
const nvidiaSmiQuery = "name,memory.total,memory.used,driver_version,uuid,pci.bus_id,temperature.gpu,utilization.gpu"

func CollectGPUs() []GPUInfo {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return nil
	}
	out, err := exec.Command("nvidia-smi",
		"--query-gpu="+nvidiaSmiQuery,
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return nil
	}

	var gpus []GPUInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		g := parseGPU(line)
		if g.Name != "" {
			gpus = append(gpus, g)
		}
	}
	return gpus
}

func parseGPU(line string) GPUInfo {
	fields := strings.Split(strings.TrimSpace(line), ",")
	if len(fields) < 8 {
		return GPUInfo{}
	}

	name := strings.TrimSpace(fields[0])
	memTotalMiB, _ := strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64)
	memUsedMiB, _ := strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 64)
	driver := strings.TrimSpace(fields[3])
	uuid := strings.TrimSpace(fields[4])
	busID := strings.TrimSpace(fields[5])
	tempC, _ := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64)
	util, _ := strconv.Atoi(strings.TrimSpace(fields[7]))

	return GPUInfo{
		Name:           name,
		Vendor:         "NVIDIA",
		MemoryTotal:    memTotalMiB * 1024 * 1024,
		MemoryUsed:     memUsedMiB * 1024 * 1024,
		DriverVersion:  driver,
		UUID:           uuid,
		BusID:          busID,
		TemperatureC:   tempC,
		UtilizationGPU: util,
	}
}