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
// 优先使用各厂商专用工具（nvidia-smi、rocm-smi），fallback 到 sysfs + lspci。
func CollectGPUs() []GPUInfo {
	var gpus []GPUInfo
	seen := make(map[string]bool)

	// 1. NVIDIA via nvidia-smi（最完整）
	for _, g := range collectNVIDIA() {
		key := g.UUID
		if !seen[key] {
			seen[key] = true
			gpus = append(gpus, g)
		}
	}

	// 2. AMD via rocm-smi
	for _, g := range collectAMD() {
		key := fmt.Sprintf("%s:%s", g.Vendor, g.Name)
		if !seen[key] {
			seen[key] = true
			gpus = append(gpus, g)
		}
	}

	// 3. Sysfs fallback（覆盖 Intel、AMD without ROCm 等）
	for _, g := range collectSysfs() {
		key := fmt.Sprintf("%s:%s:%s", g.Vendor, g.Name, g.BusID)
		if !seen[key] && g.Name != "" {
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
	return collectROCmGPUs()
}

func collectROCmGPUs() []GPUInfo {
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

	// 处理最后一个 GPU（可能没有分隔符）
	if current.Name != "" {
		current.Vendor = "AMD"
		gpus = append(gpus, current)
	}

	return gpus
}

// ========== Sysfs Fallback ==========

func collectSysfs() []GPUInfo {
	cards, err := filepath.Glob("/sys/class/drm/card[0-9]*")
	if err != nil || len(cards) == 0 {
		return nil
	}

	var gpus []GPUInfo
	for _, cardPath := range cards {
		devicePath := filepath.Join(cardPath, "device")
		if _, err := os.Stat(devicePath); os.IsNotExist(err) {
			continue
		}

		vendorID := readHexUint16(filepath.Join(devicePath, "vendor"))
		deviceID := readHexUint16(filepath.Join(devicePath, "device"))

		// 跳过 NVIDIA（已由 nvidia-smi 处理）
		if vendorID == VendorNVIDIA {
			continue
		}

		vendorName := getVendorName(vendorID)
		deviceName := getPCIDeviceName(devicePath, vendorID, deviceID)

		gpu := GPUInfo{
			Vendor: vendorName,
			Name:   deviceName,
			BusID:  filepath.Base(devicePath),
		}

		// 读取显存
		gpu.MemoryTotal = readMemoryFromSysfs(devicePath)

		// 读取温度
		gpu.TemperatureC = readGPUTemperature(cardPath)

		if gpu.Name != "" || gpu.MemoryTotal > 0 {
			gpus = append(gpus, gpu)
		}
	}

	return gpus
}

func getVendorName(vendorID uint16) string {
	switch vendorID {
	case VendorAMD:
		return "AMD"
	case VendorIntel:
		return "Intel"
	default:
		return fmt.Sprintf("0x%04x", vendorID)
	}
}

func getPCIDeviceName(devicePath string, vendorID, deviceID uint16) string {
	// 方法 1：尝试 lspci
	name := lspciGetDeviceName(devicePath)
	if name != "" {
		return name
	}

	// 方法 2：从 sysfs 读取
	if name = readSysfsName(devicePath); name != "" {
		return name
	}

	// 方法 3：生成默认名称
	return fmt.Sprintf("%s GPU %04x:%04x", getVendorName(vendorID), vendorID, deviceID)
}

func lspciGetDeviceName(devicePath string) string {
	busID := filepath.Base(devicePath)
	cmd := exec.Command("lspci", "-nn", "-s", busID)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	// 解析格式：xx:xx.x VGA compatible controller: NVIDIA GP102 [GeForce GTX 1080] (rev a1) [10de:1b06]
	line := strings.TrimSpace(string(out))
	if line == "" {
		return ""
	}

	// 取括号前的部分作为设备名
	if idx := strings.LastIndex(line, " ["); idx > 0 {
		return strings.TrimSpace(line[:idx])
	}
	return strings.TrimSpace(line)
}

func readSysfsName(devicePath string) string {
	// 尝试从 dmi/id/product_name 或类似路径获取
	namePath := filepath.Join(devicePath, "dmi", "id", "product_name")
	if content, err := os.ReadFile(namePath); err == nil {
		if name := strings.TrimSpace(string(content)); name != "" {
			return name
		}
	}
	return ""
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

func parseMemorySize(s string) uint64 {
	s = strings.TrimSpace(s)
	// 格式如 "16384 MB" 或 "8192 MiB"
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
		return val // 假设是字节
	}
}

func readMemoryFromSysfs(devicePath string) uint64 {
	// AMD：/sys/bus/pci/devices/<bus>/mem_info_vram_total
	amdPath := filepath.Join(devicePath, "mem_info_vram_total")
	if content, err := os.ReadFile(amdPath); err == nil {
		if val, _ := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64); val > 0 {
			return val
		}
	}

	// Intel：/sys/bus/pci/devices/<bus>/resource0（BAR0 是显存）
	resPath := filepath.Join(devicePath, "resource")
	if content, err := os.ReadFile(resPath); err == nil {
		if len(content) > 0 {
			// resource 文件大小即 BAR 大小
			return uint64(len(content))
		}
	}

	return 0
}

func readGPUTemperature(cardPath string) float64 {
	// 尝试从 hwmon 读取
	hwmonPaths, _ := filepath.Glob(filepath.Join(cardPath, "hwmon/hwmon*"))
	for _, hwmon := range hwmonPaths {
		tempPath := filepath.Join(hwmon, "temp1_input")
		if content, err := os.ReadFile(tempPath); err == nil {
			// hwmon 温度单位是毫摄氏度
			val, _ := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
			if val > 0 {
				return float64(val) / 1000.0
			}
		}
	}
	return 0
}
