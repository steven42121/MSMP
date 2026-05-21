package common

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type AgentInfo struct {
	UUID         string `json:"uuid"`
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`
	Arch         string `json:"arch"`
	IP           string `json:"ip"`
	PublicIP     string `json:"public_ip"`
	CPUModel     string `json:"cpu_model"`
	CPUCores     int    `json:"cpu_cores"`
	MemoryTotal  uint64 `json:"memory_total"`
	DiskTotal    uint64 `json:"disk_total"`
	AgentVersion string `json:"agent_version"`
}

type MetricData struct {
	UUID       string  `json:"uuid"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	MemUsed    uint64  `json:"mem_used"`
	MemTotal   uint64  `json:"mem_total"`
	DiskUsed   uint64  `json:"disk_used"`
	DiskTotal  uint64  `json:"disk_total"`
	NetRxBps   uint64  `json:"net_rx_bps"`
	NetTxBps   uint64  `json:"net_tx_bps"`
	Load1      float64 `json:"load1"`
	Load5      float64 `json:"load5"`
	Load15     float64 `json:"load15"`
	UptimeSec  uint64  `json:"uptime_sec"`
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

func MainLoop(serverURL, uuid, agentToken string) {
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

	if err := Register(serverURL, regData); err != nil {
		log.Printf("Register error: %v", err)
	}

	if err := ReportAssets(serverURL, assetInfo); err != nil {
		log.Printf("Asset report error: %v", err)
	}

	lastAssetsReport := time.Now()
	lastMetricsReport := time.Now()

	for {
		now := time.Now()
		ip := GetLocalIP()

		hb := HeartbeatData{
			UUID:         uuid,
			AgentVersion: AgentVersion,
			IP:           ip,
		}
		if err := Heartbeat(serverURL, hb); err != nil {
			log.Printf("Heartbeat error: %v", err)
		}

		// 资产上报（每 5 分钟）
		if now.Sub(lastAssetsReport) >= 5*time.Minute {
			newAsset := CollectAssetInfoFull()
			newAsset.AgentVersion = AgentVersion
			if err := ReportAssets(serverURL, newAsset); err != nil {
				log.Printf("Asset report error: %v", err)
			}
			lastAssetsReport = now
		}

		// 指标上报（每 60 秒）
		if now.Sub(lastMetricsReport) >= 60*time.Second {
			metrics := CollectMetrics()
			metrics.UUID = uuid
			if err := ReportMetrics(serverURL, metrics); err != nil {
				log.Printf("Metrics report error: %v", err)
			}
			lastMetricsReport = now
		}

		time.Sleep(30 * time.Second)
	}
}