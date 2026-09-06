package common

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AgentInfo struct {
	UUID              string            `json:"uuid"`
	Hostname          string            `json:"hostname"`
	OS                string            `json:"os"`
	OSVersion         string            `json:"os_version"`
	Arch              string            `json:"arch"`
	IP                string            `json:"ip"`
	PublicIP          string            `json:"public_ip"`
	CPUModel          string            `json:"cpu_model"`
	CPUCores          int               `json:"cpu_cores"`
	MemoryTotal       uint64            `json:"memory_total"`
	DiskTotal         uint64            `json:"disk_total"`
	AgentVersion      string            `json:"agent_version"`
	DiskPartitions    []DiskPartition   `json:"disk_partitions"`
	NetworkInterfaces []NetInterface    `json:"network_interfaces"`
	Processes         []ProcessInfo     `json:"processes"`
	GPUs              []GPUInfo         `json:"gpus"`
	Temperatures      []TemperatureInfo `json:"temperatures"`
}

type DiskPartition struct {
	Device     string `json:"device"`
	Mountpoint string `json:"mountpoint"`
	Fstype     string `json:"fstype"`
	Total      uint64 `json:"total"`
	Used       uint64 `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

type NetInterface struct {
	Name      string   `json:"name"`
	IP        string   `json:"ip"`
	Mac       string   `json:"mac"`
	BytesRecv uint64   `json:"bytes_recv"`
	BytesSent uint64   `json:"bytes_sent"`
}

type ProcessInfo struct {
	PID         int32   `json:"pid"`
	Name        string  `json:"name"`
	Username    string  `json:"username"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemPercent  float64 `json:"mem_percent"`
}

type GPUInfo struct {
	Name           string  `json:"name"`
	Vendor         string  `json:"vendor"`
	MemoryTotal    uint64  `json:"memory_total"`
	MemoryUsed     uint64  `json:"memory_used"`
	DriverVersion  string  `json:"driver_version"`
	UUID           string  `json:"uuid"`
	BusID          string  `json:"bus_id"`
	TemperatureC   float64 `json:"temperature_c"`
	UtilizationGPU int     `json:"utilization_gpu"`
}

type TemperatureInfo struct {
	SensorKey string  `json:"sensor_key"`
	Temp      float64 `json:"temp"`
	High      float64 `json:"high,omitempty"`
	Critical  float64 `json:"critical,omitempty"`
}

type MetricData struct {
	UUID            string  `json:"uuid"`
	CPUPercent      float64 `json:"cpu_percent"`
	MemPercent      float64 `json:"mem_percent"`
	MemUsed         uint64  `json:"mem_used"`
	MemTotal        uint64  `json:"mem_total"`
	SwapUsed        uint64  `json:"swap_used"`
	SwapTotal       uint64  `json:"swap_total"`
	DiskUsed        uint64  `json:"disk_used"`
	DiskTotal       uint64  `json:"disk_total"`
	DiskReadBytes   uint64  `json:"disk_read_bytes"`
	DiskWriteBytes  uint64  `json:"disk_write_bytes"`
	NetRxBps        uint64  `json:"net_rx_bps"`
	NetTxBps        uint64  `json:"net_tx_bps"`
	NetPktsRecv     uint64  `json:"net_pkts_recv"`
	NetPktsSent     uint64  `json:"net_pkts_sent"`
	ProcessCount    int     `json:"process_count"`
	Load1           float64 `json:"load1"`
	Load5           float64 `json:"load5"`
	Load15          float64 `json:"load15"`
	UptimeSec       uint64  `json:"uptime_sec"`
}

type HeartbeatData struct {
	UUID         string `json:"uuid"`
	AgentVersion string `json:"agent_version"`
	IP           string `json:"ip"`
}

type UpgradeInfo struct {
	Available bool   `json:"available"`
	Latest    string `json:"latest"`
	URL       string `json:"url"`
}

type HeartbeatResponse struct {
	Status  string      `json:"status"`
	Upgrade *UpgradeInfo `json:"upgrade,omitempty"`
}

type RegisterData struct {
	UUID         string `json:"uuid"`
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`
	Arch         string `json:"arch"`
	AgentVersion string `json:"agent_version"`
	AgentToken   string `json:"agent_token"`
}

const AgentVersion = "0.1.0"

func GetEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func Register(serverURL string, info RegisterData) error {
	data, _ := json.Marshal(info)
	resp, err := http.Post(serverURL+"/api/agents/register", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Register returned status: %d", resp.StatusCode)
	}
	return nil
}

func Heartbeat(serverURL string, data HeartbeatData) (*HeartbeatResponse, error) {
	body, _ := json.Marshal(data)
	resp, err := http.Post(serverURL+"/api/agents/heartbeat", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var hbResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&hbResp); err != nil {
		return nil, err
	}
	return &hbResp, nil
}

// SelfUpdate 从指定 URL 下载新版 Agent 并替换当前进程可执行文件。
// 仅在非容器环境且 URL 非空时执行。
func SelfUpdate(downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("download URL 为空")
	}
	// 容器内直接退出，由编排层替换镜像
	if _, err := os.Stat("/.dockerenv"); err == nil {
		log.Printf("Self-update skipped in container environment")
		return nil
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载返回 HTTP %d", resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}
	defer gz.Close()

	tarReader := tar.NewReader(gz)
	var binData []byte
	var binName string
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取压缩包失败: %w", err)
		}
		// 只提取可执行文件（不包含目录或 config 等）
		if hdr.Typeflag == tar.TypeReg && strings.Contains(hdr.Name, "msmp-agent") {
			binName = filepath.Base(hdr.Name)
			binData, err = io.ReadAll(tarReader)
			if err != nil {
				return fmt.Errorf("读取二进制失败: %w", err)
			}
			break
		}
	}

	if len(binData) == 0 {
		return fmt.Errorf("压缩包中未找到 msmp-agent 二进制")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取当前路径失败: %w", err)
	}
	currentDir := filepath.Dir(currentExe)
	newExe := filepath.Join(currentDir, binName)
	backupExe := currentExe + ".bak"

	// 备份当前二进制
	if err := os.Rename(currentExe, backupExe); err != nil {
		return fmt.Errorf("备份失败: %w", err)
	}

	// 写入新二进制
	if err := os.WriteFile(newExe, binData, 0755); err != nil {
		os.Rename(backupExe, currentExe) // 回滚
		return fmt.Errorf("写入失败: %w", err)
	}

	// 验证新二进制可执行
	if err := os.Rename(newExe, currentExe); err != nil {
		os.Rename(backupExe, currentExe) // 回滚
		return fmt.Errorf("替换失败: %w", err)
	}

	// 清理备份
	os.Remove(backupExe)

	log.Printf("Self-update complete: %s -> %s, restarting...", AgentVersion, binName)
	os.Exit(0)
	return nil
}

func ReportAssets(serverURL string, info AgentInfo) error {
	data, _ := json.Marshal(info)
	resp, err := http.Post(serverURL+"/api/agents/assets", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func ReportMetrics(serverURL string, data MetricData) error {
	body, _ := json.Marshal(data)
	resp, err := http.Post(serverURL+"/api/agents/metrics", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// MainLoop 启动 Agent 主循环，支持多节点路由。
func MainLoop(router *ClusterRouter, uuid, agentToken string) {
	hostname, _ := os.Hostname()
	regData := RegisterData{
		UUID:         uuid,
		Hostname:     hostname,
		AgentVersion: AgentVersion,
		AgentToken:   agentToken,
	}

	// 首次采集完整资产
	assetInfo := CollectAssetInfoFull()
	regData.OS = assetInfo.OS
	regData.OSVersion = assetInfo.OSVersion
	regData.Arch = assetInfo.Arch

	firstURL := router.NextNode()
	if err := Register(firstURL, regData); err != nil {
		log.Printf("Register error: %v", err)
	}

	if err := ReportAssets(firstURL, assetInfo); err != nil {
		log.Printf("Asset report error: %v", err)
	}

	lastAssetsReport := time.Now()
	lastMetricsReport := time.Now()
	lastTaskPoll := time.Now()

	for {
		now := time.Now()
		ip := GetLocalIP()
		url := router.NextNode()

		hbData := HeartbeatData{
			UUID:         uuid,
			AgentVersion: AgentVersion,
			IP:           ip,
		}
		hbResp, err := Heartbeat(url, hbData)
		if err != nil {
			log.Printf("Heartbeat error: %v", err)
			router.RecordFailure()
		} else {
			router.RecordSuccess()
			if hbResp.Upgrade != nil && hbResp.Upgrade.Available {
				log.Printf("Upgrade available: %s -> %s", AgentVersion, hbResp.Upgrade.Latest)
				if err := SelfUpdate(hbResp.Upgrade.URL); err != nil {
					log.Printf("Self-update failed: %v", err)
				}
			}
		}

		// 资产上报（每 5 分钟）
		if now.Sub(lastAssetsReport) >= 5*time.Minute {
			newAsset := CollectAssetInfoFull()
			newAsset.AgentVersion = AgentVersion
			if err := ReportAssets(url, newAsset); err != nil {
				log.Printf("Asset report error: %v", err)
				router.RecordFailure()
			} else {
				router.RecordSuccess()
			}
			lastAssetsReport = now
		}

		// 指标上报（每 60 秒）
		if now.Sub(lastMetricsReport) >= 60*time.Second {
			metrics := CollectMetrics()
			metrics.UUID = uuid
			if err := ReportMetrics(url, metrics); err != nil {
				log.Printf("Metrics report error: %v", err)
				router.RecordFailure()
			} else {
				router.RecordSuccess()
			}
			lastMetricsReport = now
		}

		// 任务轮询（每 10 秒）
		if now.Sub(lastTaskPoll) >= 10*time.Second {
			if task, err := FetchNextTask(url, uuid, agentToken); err != nil {
				log.Printf("Fetch task error: %v", err)
				router.RecordFailure()
			} else if task != nil {
				runTask(url, uuid, agentToken, task)
				router.RecordSuccess()
			}
			lastTaskPoll = now
		}

		time.Sleep(30 * time.Second)
	}
}

func runTask(serverURL, uuid, agentToken string, task *AgentTask) {
	log.Printf("Task %d received: type=%s", task.ID, task.Type)
	var result string
	var status string
	if task.Type == "shell" || task.Type == "flush_caches" {
		out, err := ExecuteCommand(task.Command, task.TimeoutSec)
		if err != nil {
			status = "failed"
			result = fmt.Sprintf("%s\nerror: %v", out, err)
		} else {
			status = "success"
			result = out
		}
	} else {
		status = "failed"
		result = fmt.Sprintf("unsupported task type: %s", task.Type)
	}
	if err := ReportTaskResult(serverURL, task.ID, agentToken, status, result); err != nil {
		log.Printf("Report task result error: %v", err)
	}
}
