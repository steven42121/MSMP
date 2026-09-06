// Package controllers contains alert engineering logic: suppression, silencing, escalation.
package controllers

import (
	"fmt"
	"log"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
)

// IsAlertSuppressed 检查告警是否被抑制（相同规则在时间窗口内已触发）
func IsAlertSuppressed(hostID uint, metric string, level string, windowMinutes int) bool {
	if windowMinutes <= 0 {
		windowMinutes = 30 // 默认30分钟
	}
	since := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)

	var count int64
	db.DB.Model(&models.HostEvent{}).
		Where("host_id = ? AND type = 'alert' AND message LIKE ? AND level = ? AND created_at > ?",
			hostID, "%"+metric+"%", level, since).
		Count(&count)
	return count > 0
}

// IsAlertSilenced 检查告警是否被静默
func IsAlertSilenced(hostID uint, labelKey, labelValue, level string) bool {
	now := time.Now()
	var count int64
	query := db.DB.Model(&models.AlertSilence{}).
		Where("end_at > ? AND (host_id = 0 OR host_id = ?) AND deleted_at IS NULL", now, hostID)

	if labelKey != "" && labelValue != "" {
		// 检查标签匹配
		query = query.Where("(label_key = ? AND label_value = ?) OR label_key = ''", labelKey, labelValue)
	}
	if level != "" {
		query = query.Where("level = ? OR level = ''", level)
	}
	query.Count(&count)
	return count > 0
}

// CreateSuppressedAlert 创建被抑制的记录（type=alert_suppressed，不触发再次抑制检查）。
func CreateSuppressedAlert(host models.Host, metric string, level string, message string) {
	db.DB.Create(&models.HostEvent{
		TenantID: host.TenantID,
		HostID:   host.ID,
		Type:     "alert_suppressed",
		Level:    level,
		Message:  message + " [suppressed]",
	})
}

// CreateSilencedAlert 创建被静默的告警记录
func CreateSilencedAlert(host models.Host, level string, message string) {
	db.DB.Create(&models.HostEvent{
		TenantID: host.TenantID,
		HostID:   host.ID,
		Type:     "alert",
		Level:    level,
		Message:  message + " [silenced]",
	})
}

// EvaluateAlertsWithEngineering 增强版告警评估（含抑制/静默）
func EvaluateAlertsWithEngineering(host models.Host, sample models.MetricSample) {
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

	for _, rule := range rules {
		value, ok := metricValues[rule.Metric]
		if !ok || !matchOperator(rule.Operator, value, rule.Threshold) {
			continue
		}

		message := fmt.Sprintf("%s %s %.1f", rule.Metric, rule.Operator, rule.Threshold)

		// 检查抑制
		var suppress models.AlertSuppression
		db.DB.Where("tenant_id = ? AND host_id = ? AND metric = ? AND level = ? AND enabled = ?",
			host.TenantID, host.ID, rule.Metric, rule.Level, true).
			First(&suppress)

		windowMinutes := 30 // 默认
		if suppress.ID != 0 {
			windowMinutes = suppress.WindowMinutes
		} else {
			// 全局抑制规则
			db.DB.Where("tenant_id = ? AND host_id = 0 AND metric = ? AND level = ? AND enabled = ?",
				host.TenantID, rule.Metric, rule.Level, true).
				First(&suppress)
			if suppress.ID != 0 {
				windowMinutes = suppress.WindowMinutes
			}
		}

		if IsAlertSuppressed(host.ID, rule.Metric, rule.Level, windowMinutes) {
			CreateSuppressedAlert(host, rule.Metric, rule.Level, message)
			continue
		}

		// 检查静默
		if IsAlertSilenced(host.ID, "", "", rule.Level) {
			CreateSilencedAlert(host, rule.Level, message)
			continue
		}

		// 创建告警
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

// StartEscalationChecker 启动告警升级检查器
func StartEscalationChecker() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		checkEscalations()
	}
}

// checkEscalations 检查需要升级的未确认告警
func checkEscalations() {
	var escalations []models.AlertEscalation
	db.DB.Where("enabled = ?", true).Find(&escalations)

	if len(escalations) == 0 {
		return
	}

	for _, esc := range escalations {
		threshold := time.Now().Add(-time.Duration(esc.NotifyAfterMin) * time.Minute)

		// 查找未确认且超过阈值的告警（精确匹配 event.ID，防止 ALERT #12 误匹配 ALERT #123）
		var events []models.HostEvent
		db.DB.Where("tenant_id = ? AND type = 'alert' AND level = ? AND acknowledged = false AND created_at < ?",
			esc.TenantID, esc.TriggerLevel, threshold).
			Order("created_at ASC").
			Limit(10).
			Find(&events)

		for _, event := range events {
			// 检查是否已升级过（精确匹配事件 ID）
			var upgradeCount int64
			db.DB.Model(&models.HostEvent{}).
				Where("host_id = ? AND type = 'escalation' AND message = ?",
					event.HostID,
					fmt.Sprintf("ALERT #%d ESCALATED", event.ID)).
				Count(&upgradeCount)

			if upgradeCount >= int64(esc.RetryCount) {
				continue
			}

			newLevel := escalateLevel(event.Level)
			// 创建升级事件
			upgradeMsg := fmt.Sprintf("ALERT #%d ESCALATED (level: %s -> %s)",
				event.ID, event.Level, newLevel)
			upgradeEvent := models.HostEvent{
				TenantID: event.TenantID,
				HostID:   event.HostID,
				Type:     "escalation",
				Level:    "critical",
				Message:  upgradeMsg,
			}
			db.DB.Create(&upgradeEvent)

			// 关联原始告警
			db.DB.Model(&event).Update("acknowledged", true)

			log.Printf("[Alert] Escalated alert #%d for host #%d", event.ID, event.HostID)
			notifyWebhook(upgradeEvent, hostFromID(event.HostID))
		}
	}
}

// escalateLevel 将告警级别升级
func escalateLevel(level string) string {
	switch level {
	case "warning":
		return "critical"
	case "info":
		return "warning"
	default:
		return level
	}
}

// hostFromID 根据 host_id 查询主机。
func hostFromID(hostID uint) models.Host {
	var h models.Host
	db.DB.Where("id = ?", hostID).First(&h)
	return h
}
