package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

func Heartbeat(serverURL string, data HeartbeatData) error {
	body, _ := json.Marshal(data)
	resp, err := http.Post(serverURL+"/api/agents/heartbeat", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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

		hb := HeartbeatData{
			UUID:         uuid,
			AgentVersion: AgentVersion,
			IP:           ip,
		}
		if err := Heartbeat(url, hb); err != nil {
			log.Printf("Heartbeat error: %v", err)
			router.RecordFailure()
		} else {
			router.RecordSuccess()
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
	if task.Type == "shell" {
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
