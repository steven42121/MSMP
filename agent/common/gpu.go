package common

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const nvidiaSmiQuery = "name,memory.total,memory.used,driver_version,uuid,pci.bus_id,temperature.gpu,utilization.gpu"

// Vendor IDs
const (
	VendorNVIDIA uint16 = 0x10de
	VendorAMD    uint16 = 0x1002
	VendorIntel  uint16 = 0x8086
)

// CollectGPUs 采集所有 GPU 信息（NVIDIA、AMD、Intel 等）。
func CollectGPUs() []GPUInfo {
	var gpus []GPUInfo
	seen := make(map[string]bool)

	// 1. NVIDIA via nvidia-smi
	for _, g := range collectNVIDIA() {
		key := g.UUID
		if !seen[key] {
			seen[key] = true
			gpus = append(gpus, g)
		}
	}

	// 2. AMD via rocm-smi
	for _, g := range collectAMD() {
		key := fmt.Sprintf("AMD:%s", g.Name)
		if !seen[key] {
			seen[key] = true
			gpus = append(gpus, g)
		}
	}

	// 3. Intel/其他 via oneAPI 风格采集（sysfs + lspci）
	for _, g := range collectIntelGPUs() {
		key := fmt.Sprintf("%s:%s", g.Vendor, g.Name)
		if !seen[key] {
			seen[key] = true
			gpus = append(gpus, g)
		}
	}

	return gpus
}

// ========== NVIDIA ==========

func collectNVIDIA() []GPUInfo {
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
		g := parseNVIDIAGPU(line)
		if g.Name != "" {
			gpus = append(gpus, g)
		}
	}
	return gpus
}

func parseNVIDIAGPU(line string) GPUInfo {
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

// ========== AMD ==========

func collectAMD() []GPUInfo {
	if _, err := exec.LookPath("rocm-smi"); err != nil {
		return nil
	}
	out, err := exec.Command("rocm-smi", "--showallinfo").Output()
	if err != nil {
		return nil
	}
	return parseROCmOutput(string(out))
}

func parseROCmOutput(output string) []GPUInfo {
	var gpus []GPUInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	var current GPUInfo
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "GPU part number") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.Name = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "GPU memory size") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.MemoryTotal = parseMemorySize(strings.TrimSpace(parts[1]))
			}
		} else if strings.HasPrefix(line, "GPU temperature") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				temp, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				current.TemperatureC = temp
			}
		} else if strings.HasPrefix(line, "Unique board identifier") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.UUID = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "PCI bus ID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.BusID = strings.TrimSpace(parts[1])
			}
		} else if line == "---" {
			if current.Name != "" {
				current.Vendor = "AMD"
				gpus = append(gpus, current)
			}
			current = GPUInfo{}
		}
	}

	if current.Name != "" {
		current.Vendor = "AMD"
		gpus = append(gpus, current)
	}

	return gpus
}

// ========== Intel (oneAPI 风格) ==========

func collectIntelGPUs() []GPUInfo {
	// 方式 1: lspci 直接查询 Intel VGA/3D 控制器
	if gpus := collectFromLspci(); len(gpus) > 0 {
		return gpus
	}

	// 方式 2: sysfs fallback
	return collectFromSysfs()
}

func collectFromLspci() []GPUInfo {
	cmd := exec.Command("lspci", "-nn", "-d", "8086:")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var gpus []GPUInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.Contains(line, "VGA") || strings.Contains(line, "3D") || strings.Contains(line, "Display") {
			gpu := parseLSPCIGPU(line)
			if gpu.Name != "" {
				gpus = append(gpus, gpu)
			}
		}
	}
	return gpus
}

func parseLSPCIGPU(line string) GPUInfo {
	// 格式：00:02.0 VGA compatible controller: Intel Corporation Device XXXX (rev XX) [8086:XXXX]
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) != 2 {
		return GPUInfo{}
	}

	busID := strings.TrimSpace(parts[0])
	desc := strings.TrimSpace(parts[1])

	// 提取设备名（去掉 VGA/3D 前缀和厂商 ID 部分）
	deviceName := extractDeviceName(desc)

	// 读取 BAR 信息（显存大小）
	memTotal := readGPUMemoryFromSysfs(busID)

	// 读取温度
	tempC := readGPUTemperatureFromSysfs(busID)

	return GPUInfo{
		Vendor:       "Intel",
		Name:         deviceName,
		BusID:        busID,
		MemoryTotal:  memTotal,
		TemperatureC: tempC,
	}
}

func extractDeviceName(desc string) string {
	// 去掉结尾的 [vendor:device] 部分
	if idx := strings.LastIndex(desc, " ["); idx > 0 {
		desc = desc[:idx]
	}
	// 去掉 VGA/3D/Display 前缀
	desc = strings.ReplaceAll(desc, "VGA compatible controller: ", "")
	desc = strings.ReplaceAll(desc, "3D controller: ", "")
	desc = strings.ReplaceAll(desc, "Display controller: ", "")
	return strings.TrimSpace(desc)
}

func collectFromSysfs() []GPUInfo {
	cards, err := filepath.Glob("/sys/class/drm/card[0-9]*")
	if err != nil || len(cards) == 0 {
		return nil
	}

	var gpus []GPUInfo
	seen := make(map[string]bool)

	for _, cardPath := range cards {
		devicePath := filepath.Join(cardPath, "device")
		if _, err := os.Stat(devicePath); os.IsNotExist(err) {
			continue
		}

		vendorID := readHexUint16(filepath.Join(devicePath, "vendor"))
		if vendorID != VendorIntel {
			continue
		}

		busID := filepath.Base(devicePath)
		key := fmt.Sprintf("Intel:%s", busID)
		if seen[key] {
			continue
		}
		seen[key] = true

		deviceName := getSysfsDeviceName(devicePath)
		memTotal := readGPUMemoryFromSysfs(busID)
		tempC := readGPUTemperatureFromSysfs(busID)

		gpus = append(gpus, GPUInfo{
			Vendor:       "Intel",
			Name:         deviceName,
			BusID:        busID,
			MemoryTotal:  memTotal,
			TemperatureC: tempC,
		})
	}

	return gpus
}

func getSysfsDeviceName(devicePath string) string {
	// 尝试从 dmi/id/product_name 读取
	namePath := filepath.Join(devicePath, "dmi", "id", "product_name")
	if content, err := os.ReadFile(namePath); err == nil {
		if name := strings.TrimSpace(string(content)); name != "" {
			return name
		}
	}

	// 从 resource 文件获取总线信息
	subsystem := readHexUint32(filepath.Join(devicePath, "subsystem_device"))
	vendor := readHexUint32(filepath.Join(devicePath, "subsystem_vendor"))

	if vendor == uint32(VendorIntel) && subsystem != 0 {
		return fmt.Sprintf("Intel Graphics (0x%04x)", subsystem)
	}

	return "Intel Integrated Graphics"
}

// ========== 辅助函数 ==========

func readHexUint16(path string) uint16 {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(content)), 16, 16)
	if err != nil {
		return 0
	}
	return uint16(val)
}

func readHexUint32(path string) uint32 {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(content)), 16, 32)
	if err != nil {
		return 0
	}
	return uint32(val)
}

func parseMemorySize(s string) uint64 {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return 0
	}
	val, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	switch {
	case strings.HasPrefix(parts[len(parts)-1], "GB"):
		return val * 1024 * 1024 * 1024
	case strings.HasPrefix(parts[len(parts)-1], "MB"):
		return val * 1024 * 1024
	case strings.HasPrefix(parts[len(parts)-1], "MiB"):
		return val * 1024 * 1024
	default:
		return val
	}
}

func readGPUMemoryFromSysfs(busID string) uint64 {
	// AMD: /sys/bus/pci/devices/<bus>/mem_info_vram_total
	amdPath := fmt.Sprintf("/sys/bus/pci/devices/%s/mem_info_vram_total", busID)
	if content, err := os.ReadFile(amdPath); err == nil {
		if val, _ := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64); val > 0 {
			return val
		}
	}

	// Intel: 从 BAR0 读取
	intelPath := fmt.Sprintf("/sys/bus/pci/devices/%s/resource0", busID)
	if content, err := os.ReadFile(intelPath); err == nil {
		if len(content) > 0 {
			return uint64(len(content))
		}
	}

	return 0
}

func readGPUTemperatureFromSysfs(busID string) float64 {
	// 方法 1: 通过 DRM card 路径
	cardPaths, _ := filepath.Glob("/sys/class/drm/card[0-9]*/device")
	for _, devicePath := range cardPaths {
		if !strings.HasSuffix(devicePath, busID) {
			continue
		}

		hwmonPaths, _ := filepath.Glob(filepath.Join(devicePath, "hwmon/hwmon*"))
		for _, hwmon := range hwmonPaths {
			tempPath := filepath.Join(hwmon, "temp1_input")
			if content, err := os.ReadFile(tempPath); err == nil {
				if val, _ := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64); val > 0 {
					return float64(val) / 1000.0
				}
			}
		}
	}

	// 方法 2: 直接搜索
	pattern := fmt.Sprintf("/sys/bus/pci/devices/%s/hwmon/hwmon*/temp*_input", busID)
	matches, _ := filepath.Glob(pattern)
	for _, path := range matches {
		if content, err := os.ReadFile(path); err == nil {
			if val, _ := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64); val > 0 {
				return float64(val) / 1000.0
			}
		}
	}

	return 0
}
