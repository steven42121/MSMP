package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"MSMP/server/db"
	"MSMP/server/models"
)

// 任务结构（未在 models 中定义，使用动态表或 JSON 字段）
// 这里使用 HostEvent 表来存储任务（简化处理，实际应该单独建表）

type TaskRequest struct {
	HostUUID    string `json:"host_uuid"`
	Type        string `json:"type"`        // shell, restart, upgrade
	Command     string `json:"command"`
	Level       string `json:"level"`       // info, warning, critical
}

func TasksHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var events []models.HostEvent
		query := db.DB.Where("tenant_id = ? AND type IN ('task','shell','restart','upgrade')", tenantID)

		if hostID := r.URL.Query().Get("host_id"); hostID != "" {
			query = query.Where("host_id = ?", hostID)
		}
		if status := r.URL.Query().Get("status"); status != "" {
			query = query.Where("level = ?", status)
		}

		var total int64
		query.Model(&models.HostEvent{}).Count(&total)

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		if page <= 0 { page = 1 }
		if pageSize <= 0 || pageSize > 100 { pageSize = 20 }

		query.Order("created_at DESC").Offset((page-1)*pageSize).Limit(pageSize).Find(&events)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data": events, "total": total, "page": page, "page_size": pageSize,
		})

	case http.MethodPost:
		var req TaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		var host models.Host
		if err := db.DB.Where("uuid = ? AND tenant_id = ?", req.HostUUID, tenantID).First(&host).Error; err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
			return
		}

		event := models.HostEvent{
			TenantID: tenantID,
			HostID:   host.ID,
			Type:     req.Type,
			Level:    req.Level,
			Message:  req.Command,
		}
		if event.Level == "" {
			event.Level = "info"
		}
		if event.Type == "" {
			event.Type = "task"
		}
		db.DB.Create(&event)
		writeJSON(w, http.StatusCreated, event)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func TaskDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/tasks/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id required"})
		return
	}

	taskID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid task id"})
		return
	}

	var event models.HostEvent
	if err := db.DB.Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&event).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, event)
	case http.MethodDelete:
		db.DB.Delete(&event)
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}