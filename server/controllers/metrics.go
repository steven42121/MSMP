package controllers

import (
	"net/http"
	"strconv"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
)

// MetricsHandler 监控数据查询
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tenantID := getTenantID(r)
	hostUUID := r.URL.Query().Get("host_uuid")
	if hostUUID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host_uuid is required"})
		return
	}

	var host models.Host
	if err := db.DB.Where("uuid = ? AND tenant_id = ?", hostUUID, tenantID).First(&host).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
		return
	}

	// 默认查询最近 24 小时
	duration := r.URL.Query().Get("duration")
	dur, err := time.ParseDuration(duration)
	if err != nil || duration == "" {
		dur = 24 * time.Hour
	}

	since := time.Now().Add(-dur)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 300
	}

	var metrics []models.MetricSample
	db.DB.Where("host_id = ? AND timestamp >= ?", host.ID, since).
		Order("timestamp ASC").Limit(limit).Find(&metrics)

	// 窗口内无数据时回退到最近 limit 条（覆盖离线主机：最后一批指标已超出查询窗口）
	if len(metrics) == 0 {
		db.DB.Where("host_id = ?", host.ID).
			Order("timestamp DESC").Limit(limit).Find(&metrics)
		// 反转为时间升序，保持前端曲线从左到右递增
		for i, j := 0, len(metrics)-1; i < j; i, j = i+1, j-1 {
			metrics[i], metrics[j] = metrics[j], metrics[i]
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"host_uuid": hostUUID,
		"duration":  dur.String(),
		"data":      metrics,
	})
}