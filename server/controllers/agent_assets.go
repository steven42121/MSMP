package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"MSMP/server/config"
	"MSMP/server/db"
	"MSMP/server/models"
)

type AgentRegisterRequest struct {
	UUID         string `json:"uuid"`
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`
	Arch         string `json:"arch"`
	AgentVersion string `json:"agent_version"`
	AgentToken   string `json:"agent_token"`
}

type AgentRegisterResponse struct {
	Success bool   `json:"success"`
	HostID  uint   `json:"host_id,omitempty"`
	Message string `json:"message"`
}

type AssetInfo struct {
	UUID              string             `json:"uuid"`
	Hostname          string             `json:"hostname"`
	OS                string             `json:"os"`
	OSVersion         string             `json:"os_version"`
	Arch              string             `json:"arch"`
	IP                string             `json:"ip"`
	PublicIP          string             `json:"public_ip"`
	CPUModel          string             `json:"cpu_model"`
	CPUCores          int                `json:"cpu_cores"`
	MemoryTotal       uint64             `json:"memory_total"`
	DiskTotal         uint64             `json:"disk_total"`
	AgentVersion      string             `json:"agent_version"`
	DiskPartitions    []json.RawMessage  `json:"disk_partitions"`
	NetworkInterfaces []json.RawMessage  `json:"network_interfaces"`
	Processes         []json.RawMessage  `json:"processes"`
	GPUs              []json.RawMessage  `json:"gpus"`
	Temperatures      []json.RawMessage  `json:"temperatures"`
}

type MetricInfo struct {
	UUID         string  `json:"uuid"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemPercent   float64 `json:"mem_percent"`
	MemUsed      uint64  `json:"mem_used"`
	MemTotal     uint64  `json:"mem_total"`
	SwapUsed     uint64  `json:"swap_used"`
	SwapTotal    uint64  `json:"swap_total"`
	DiskUsed     uint64  `json:"disk_used"`
	DiskTotal    uint64  `json:"disk_total"`
	DiskReadBytes uint64 `json:"disk_read_bytes"`
	DiskWriteBytes uint64 `json:"disk_write_bytes"`
	NetRxBps     uint64  `json:"net_rx_bps"`
	NetTxBps     uint64  `json:"net_tx_bps"`
	NetPktsRecv  uint64  `json:"net_pkts_recv"`
	NetPktsSent  uint64  `json:"net_pkts_sent"`
	ProcessCount int     `json:"process_count"`
	Load1        float64 `json:"load1"`
	Load5        float64 `json:"load5"`
	Load15       float64 `json:"load15"`
	UptimeSec    uint64  `json:"uptime_sec"`
}

type HeartbeatRequest struct {
	UUID         string `json:"uuid"`
	AgentVersion string `json:"agent_version"`
	IP           string `json:"ip"`
}

// AgentRegisterHandler Agent 注册
func AgentRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req AgentRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.UUID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uuid is required"})
		return
	}

	// 验证 AgentToken，确定租户
	tenantID := uint(0)
	if req.AgentToken != "" {
		var token models.AgentToken
		if err := db.DB.Where("token = ? AND revoked = ?", req.AgentToken, false).First(&token).Error; err == nil {
			if token.ExpiresAt == nil || token.ExpiresAt.After(time.Now()) {
				tenantID = token.TenantID
				// 绑定 token 到 host
				if token.HostID == nil {
					var host models.Host
					if err := db.DB.Where("uuid = ?", req.UUID).First(&host).Error; err == nil {
						db.DB.Model(&token).Update("host_id", host.ID)
					}
				}
			}
		}
	}

	if tenantID == 0 {
		// 默认租户
		var tenant models.Tenant
		if err := db.DB.Where("slug = ?", "default").First(&tenant).Error; err != nil {
			tenant = models.Tenant{Name: "Default", Slug: "default"}
			db.DB.Create(&tenant)
		}
		tenantID = tenant.ID
	}

	// 查找或创建主机
	var host models.Host
	result := db.DB.Where("uuid = ?", req.UUID).First(&host)
	if result.Error != nil {
		now := time.Now()
		host = models.Host{
			TenantID:      tenantID,
			UUID:          req.UUID,
			Hostname:      req.Hostname,
			OS:            req.OS,
			OSVersion:     req.OSVersion,
			Arch:          req.Arch,
			AgentVersion:  req.AgentVersion,
			Status:        "online",
			LastHeartbeat: &now,
			RegisteredAt:  now,
		}
		if err := db.DB.Create(&host).Error; err != nil {
			log.Printf("Failed to create host: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to register host"})
			return
		}
		log.Printf("New agent registered: %s (%s)", req.Hostname, req.UUID)
	} else {
		now := time.Now()
		updates := map[string]interface{}{
			"hostname":       req.Hostname,
			"os":             req.OS,
			"os_version":     req.OSVersion,
			"arch":           req.Arch,
			"agent_version":  req.AgentVersion,
			"status":         "online",
			"last_heartbeat": now,
		}
		db.DB.Model(&host).Updates(updates)
	}

	writeJSON(w, http.StatusOK, AgentRegisterResponse{
		Success: true,
		HostID:  host.ID,
		Message: "registered",
	})
}

type HeartbeatResponse struct {
	Status string      `json:"status"`
	Upgrade *UpgradeInfo `json:"upgrade,omitempty"`
}

type UpgradeInfo struct {
	Available bool   `json:"available"`
	Latest    string `json:"latest"`
	URL       string `json:"url"`
}

// AgentHeartbeatHandler Agent 心跳
func AgentHeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.UUID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uuid is required"})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":         "online",
		"last_heartbeat": now,
	}
	if req.AgentVersion != "" {
		updates["agent_version"] = req.AgentVersion
	}
	if req.IP != "" {
		updates["ip"] = req.IP
	}

	result := db.DB.Model(&models.Host{}).Where("uuid = ?", req.UUID).Updates(updates)
	if result.RowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found", "hint": "register first"})
		return
	}

	// 检查是否需要升级
	var upgrade *UpgradeInfo
	c := config.C
	if req.AgentVersion != "" && c.Agent.LatestVersion != "" {
		if semverLess(req.AgentVersion, c.Agent.LatestVersion) {
			url := replaceVersionPlaceholder(c.Agent.DownloadURL, c.Agent.LatestVersion)
			upgrade = &UpgradeInfo{
				Available: true,
				Latest:    c.Agent.LatestVersion,
				URL:       url,
			}
		}
	}

	resp := HeartbeatResponse{Status: "ok", Upgrade: upgrade}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// AgentUpgradeHandler 触发 Agent 升级，返回当前最新版信息
// 支持两种路径：/api/agents/upgrade/{uuid} 和 /api/hosts/{uuid}/agent/upgrade
func AgentUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// 解析 UUID：支持 /api/agents/upgrade/{uuid} 和 /api/hosts/{uuid}/agent/upgrade
	path := r.URL.Path
	var uuid string
	if strings.HasPrefix(path, "/api/agents/upgrade/") {
		uuid = strings.TrimPrefix(path, "/api/agents/upgrade/")
	} else if strings.Contains(path, "/agent/upgrade") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/hosts/"), "/")
		uuid = parts[0]
	}
	if uuid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uuid required"})
		return
	}

	var host models.Host
	if err := db.DB.Where("uuid = ?", uuid).First(&host).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
		return
	}

	c := config.C
	resp := UpgradeInfo{
		Latest: c.Agent.LatestVersion,
	}
	if host.AgentVersion != "" && semverLess(host.AgentVersion, c.Agent.LatestVersion) {
		resp.Available = true
		resp.URL = replaceVersionPlaceholder(c.Agent.DownloadURL, c.Agent.LatestVersion)
	} else {
		resp.Available = false
	}

	writeJSON(w, http.StatusOK, resp)
}

// semverLess 简单语义化版本比较（仅支持主版本.次版本.修订版本）
func semverLess(a, b string) bool {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		var va, vb int
		if i < len(pa) {
			fmt.Sscanf(pa[i], "%d", &va)
		}
		if i < len(pb) {
			fmt.Sscanf(pb[i], "%d", &vb)
		}
		if va < vb {
			return true
		}
		if va > vb {
			return false
		}
	}
	return false
}

// replaceVersionPlaceholder 替换 download_url 模板中的 {{.Tag}} 为实际版本号
func replaceVersionPlaceholder(template, version string) string {
	return strings.NewReplacer(
		"{{.Tag}}", version,
		"{{version}}", version,
	).Replace(template)
}

// AgentAssetReportHandler 资产上报
func AgentAssetReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var info AssetInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		log.Println("Error parsing asset info:", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid data"})
		return
	}

	if info.UUID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uuid is required"})
		return
	}

	// 更新主机资产信息
	var host models.Host
	if err := db.DB.Where("uuid = ?", info.UUID).First(&host).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
		return
	}

	updates := map[string]interface{}{
		"hostname":     info.Hostname,
		"os":           info.OS,
		"os_version":   info.OSVersion,
		"arch":         info.Arch,
		"ip":           info.IP,
		"public_ip":    info.PublicIP,
		"cpu_model":    info.CPUModel,
		"cpu_cores":    info.CPUCores,
		"memory_total": info.MemoryTotal,
		"disk_total":   info.DiskTotal,
	}
	if info.AgentVersion != "" {
		updates["agent_version"] = info.AgentVersion
	}

	db.DB.Model(&host).Updates(updates)

	// 保存资产快照
	payload, _ := json.Marshal(info)
	snapshot := models.AssetSnapshot{
		TenantID: host.TenantID,
		HostID:   host.ID,
		Payload:  string(payload),
	}
	db.DB.Create(&snapshot)

	log.Printf("Asset updated: %s (%s)", info.Hostname, info.UUID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// AgentMetricReportHandler 监控指标上报
func AgentMetricReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var info MetricInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid data"})
		return
	}

	if info.UUID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uuid is required"})
		return
	}

	var host models.Host
	if err := db.DB.Where("uuid = ?", info.UUID).First(&host).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
		return
	}

	now := time.Now()
	sample := models.MetricSample{
		TenantID:     host.TenantID,
		HostID:       host.ID,
		Timestamp:    now,
		CPUPercent:   info.CPUPercent,
		MemPercent:   info.MemPercent,
		MemUsed:      info.MemUsed,
		MemTotal:     info.MemTotal,
		SwapUsed:     info.SwapUsed,
		SwapTotal:    info.SwapTotal,
		DiskUsed:     info.DiskUsed,
		DiskTotal:    info.DiskTotal,
		DiskReadBytes: info.DiskReadBytes,
		DiskWriteBytes: info.DiskWriteBytes,
		NetRxBps:     info.NetRxBps,
		NetTxBps:     info.NetTxBps,
		NetPktsRecv:  info.NetPktsRecv,
		NetPktsSent:  info.NetPktsSent,
		ProcessCount: info.ProcessCount,
		Load1:        info.Load1,
		Load5:        info.Load5,
		Load15:       info.Load15,
		UptimeSec:    info.UptimeSec,
	}

	if err := db.DB.Create(&sample).Error; err != nil {
		log.Printf("Failed to save metric: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save metric"})
		return
	}

	// 同时更新心跳时间
	db.DB.Model(&host).Updates(map[string]interface{}{
		"status":         "online",
		"last_heartbeat": now,
	})

	EvaluateAlertsWithEngineering(host, sample)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
