// Package controllers contains alert statistics API handlers.
package controllers

import (
	"net/http"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
)

// AlertStatsHandler 返回告警统计数据
func AlertStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tenantID := getTenantID(r)

	// 最近7天告警趋势
	now := time.Now()
	days := []string{}
	countsByDay := []int{}
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
		end := start.Add(24 * time.Hour)
		days = append(days, start.Format("01-02"))

		var count int64
		db.DB.Model(&models.HostEvent{}).
			Where("tenant_id = ? AND type = 'alert' AND created_at >= ? AND created_at < ?",
				tenantID, start, end).
			Count(&count)
		countsByDay = append(countsByDay, int(count))
	}

	// 按级别统计
	var criticalCount, warningCount, infoCount int64
	db.DB.Model(&models.HostEvent{}).
		Where("tenant_id = ? AND type = 'alert' AND level = 'critical'", tenantID).
		Count(&criticalCount)
	db.DB.Model(&models.HostEvent{}).
		Where("tenant_id = ? AND type = 'alert' AND level = 'warning'", tenantID).
		Count(&warningCount)
	db.DB.Model(&models.HostEvent{}).
		Where("tenant_id = ? AND type = 'alert' AND level = 'info'", tenantID).
		Count(&infoCount)

	// 按主机统计 Top 5
	type hostAlertCount struct {
		HostID   uint   `json:"host_id"`
		Hostname string `json:"hostname"`
		Count    int64  `json:"count"`
	}
	var hostStats []hostAlertCount
	db.DB.Table("hosts h").
		Select("h.id as host_id, h.hostname, COUNT(e.id) as count").
		Joins("LEFT JOIN host_events e ON e.host_id = h.id AND e.tenant_id = h.tenant_id AND e.type = 'alert'").
		Where("h.tenant_id = ?", tenantID).
		Group("h.id, h.hostname").
		Order("count DESC").
		Limit(5).
		Scan(&hostStats)

	// 未确认告警数
	var unackCount int64
	db.DB.Model(&models.HostEvent{}).
		Where("tenant_id = ? AND type = 'alert' AND acknowledged = false", tenantID).
		Count(&unackCount)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"trend": map[string]interface{}{
			"days":   days,
			"counts": countsByDay,
		},
		"by_level": map[string]int64{
			"critical": criticalCount,
			"warning":  warningCount,
			"info":     infoCount,
		},
		"by_host": hostStats,
		"unacknowledged": unackCount,
	})
}
