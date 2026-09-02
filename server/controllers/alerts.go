package controllers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"MSMP/server/config"
	"MSMP/server/db"
	"MSMP/server/models"
)

func AlertsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tenantID := getTenantID(r)
	var events []models.HostEvent

	query := db.DB.Where("tenant_id = ? AND type IN ('alert','error','offline')", tenantID)

	if level := r.URL.Query().Get("level"); level != "" {
		query = query.Where("level = ?", level)
	}
	if hostID := r.URL.Query().Get("host_id"); hostID != "" {
		query = query.Where("host_id = ?", hostID)
	}
	if ack := r.URL.Query().Get("ack"); ack != "" {
		query = query.Where("acknowledged = ?", ack == "true")
	}

	var total int64
	query.Model(&models.HostEvent{}).Count(&total)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&events)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": events, "total": total, "page": page, "page_size": pageSize,
	})
}

// evaluateMetricAlerts 根据告警规则生成告警事件（30 分钟内同消息不重复）
func evaluateMetricAlerts(host models.Host, sample models.MetricSample) {
	diskPercent := 0.0
	if sample.DiskTotal > 0 {
		diskPercent = float64(sample.DiskUsed) / float64(sample.DiskTotal) * 100
	}

	metricValues := map[string]float64{
		"cpu":  sample.CPUPercent,
		"mem":  sample.MemPercent,
		"disk": diskPercent,
	}

	var rules []models.AlertRule
	db.DB.Where("tenant_id = ? AND enabled = ?", host.TenantID, true).Find(&rules)

	now := time.Now()
	since := now.Add(-30 * time.Minute)

	for _, rule := range rules {
		value, ok := metricValues[rule.Metric]
		if !ok || !matchOperator(rule.Operator, value, rule.Threshold) {
			continue
		}

		message := rule.Metric + " " + rule.Operator + " " + strconv.FormatFloat(rule.Threshold, 'f', 1, 64)
		var count int64
		db.DB.Model(&models.HostEvent{}).
			Where("host_id = ? AND type = ? AND level = ? AND message = ? AND created_at > ?",
				host.ID, "alert", rule.Level, message, since).
			Count(&count)
		if count > 0 {
			continue
		}

		event := models.HostEvent{
			TenantID: host.TenantID,
			HostID:   host.ID,
			Type:     "alert",
			Level:    rule.Level,
			Message:  message,
		}
		db.DB.Create(&event)
		notifyWebhook(event, host)
	}
}

func matchOperator(op string, value, threshold float64) bool {
	switch op {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	}
	return false
}

func notifyWebhook(event models.HostEvent, host models.Host) {
	url := GetSetting("notification.webhookurl")
	if url == "" {
		url = config.C.Notification.WebhookURL
	}
	if url == "" {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"event":     event,
		"hostname":  host.Hostname,
		"host_uuid": host.UUID,
	})
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("webhook notify error: %v", err)
		return
	}
	defer resp.Body.Close()
}

// StartOfflineChecker 后台检测离线主机并生成事件
func StartOfflineChecker(defaultSec int) {
	if defaultSec <= 0 {
		defaultSec = 120
	}
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		sec := defaultSec
		if v := GetSetting("agent.offlineaftersec"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				sec = n
			}
		}
		checkOfflineHosts(sec)
	}
}

func checkOfflineHosts(offlineAfterSec int) {
	threshold := time.Now().Add(-time.Duration(offlineAfterSec) * time.Second)
	var hosts []models.Host
	db.DB.Where("status = ? AND last_heartbeat < ?", "online", threshold).Find(&hosts)

	for _, host := range hosts {
		db.DB.Model(&host).Update("status", "offline")

		var count int64
		db.DB.Model(&models.HostEvent{}).
			Where("host_id = ? AND type = ? AND created_at > ?",
				host.ID, "offline", time.Now().Add(-time.Duration(offlineAfterSec)*time.Second)).
			Count(&count)
		if count == 0 {
			db.DB.Create(&models.HostEvent{
				TenantID: host.TenantID,
				HostID:   host.ID,
				Type:     "offline",
				Level:    "critical",
				Message:  "Host went offline",
			})
		}
	}
}
