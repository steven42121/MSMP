package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
)

type TaskRequest struct {
	HostUUID   string   `json:"host_uuid"`
	HostUUIDs  []string `json:"host_uuids"`
	Type       string   `json:"type"`       // shell, restart, upgrade
	Command    string   `json:"command"`
	TimeoutSec int      `json:"timeout_sec"`
}

type TaskResultRequest struct {
	Status string `json:"status"` // success, failed
	Result string `json:"result"`
}

func TasksHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var tasks []models.Task
		query := db.DB.Where("tenant_id = ?", tenantID)

		if hostID := r.URL.Query().Get("host_id"); hostID != "" {
			query = query.Where("host_id = ?", hostID)
		}
		if status := r.URL.Query().Get("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if typ := r.URL.Query().Get("type"); typ != "" {
			query = query.Where("type = ?", typ)
		}

		var total int64
		query.Model(&models.Task{}).Count(&total)

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 || pageSize > 100 {
			pageSize = 20
		}

		query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data": tasks, "total": total, "page": page, "page_size": pageSize,
		})

	case http.MethodPost:
		var req TaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		uuids := req.HostUUIDs
		if len(uuids) == 0 && req.HostUUID != "" {
			uuids = []string{req.HostUUID}
		}
		if len(uuids) == 0 || req.Command == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host_uuid(s) and command required"})
			return
		}

		typ := req.Type
		if typ == "" {
			typ = "shell"
		}

		var tasks []models.Task
		for _, uuid := range uuids {
			var host models.Host
			if err := db.DB.Where("uuid = ? AND tenant_id = ?", uuid, tenantID).First(&host).Error; err != nil {
				continue
			}
			task := models.Task{
				TenantID:   tenantID,
				HostID:     host.ID,
				Type:       typ,
				Command:    req.Command,
				Status:     "pending",
				TimeoutSec: req.TimeoutSec,
				CreatedBy:  getUserID(r),
			}
			if err := db.DB.Create(&task).Error; err == nil {
				tasks = append(tasks, task)
			}
		}

		if len(tasks) == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no valid host found"})
			return
		}
		if len(tasks) == 1 {
			writeJSON(w, http.StatusCreated, tasks[0])
		} else {
			writeJSON(w, http.StatusCreated, map[string]interface{}{"data": tasks})
		}

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

	var task models.Task
	if err := db.DB.Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&task).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, task)
	case http.MethodPut:
		var updates struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if updates.Status == "canceled" && task.Status == "pending" {
			db.DB.Model(&task).Update("status", "canceled")
		}
		db.DB.Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&task)
		writeJSON(w, http.StatusOK, task)
	case http.MethodDelete:
		db.DB.Delete(&task)
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// AgentTaskHandler 处理 /api/agents/tasks/next 和 /api/agents/tasks/{id}/result
func AgentTaskHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	path := strings.TrimPrefix(r.URL.Path, "/api/agents/tasks/")
	parts := strings.Split(path, "/")

	switch {
	case len(parts) == 1 && parts[0] == "next" && r.Method == http.MethodGet:
		hostUUID := r.URL.Query().Get("host_uuid")
		if hostUUID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host_uuid required"})
			return
		}

		var host models.Host
		if err := db.DB.Where("uuid = ? AND tenant_id = ?", hostUUID, tenantID).First(&host).Error; err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
			return
		}

		var task models.Task
		if err := db.DB.Where("host_id = ? AND status = ?", host.ID, "pending").
			Order("created_at ASC").First(&task).Error; err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "no pending task"})
			return
		}

		now := time.Now()
		db.DB.Model(&task).Updates(map[string]interface{}{
			"status":     "running",
			"started_at": now,
		})
		task.Status = "running"
		task.StartedAt = &now
		writeJSON(w, http.StatusOK, task)

	case len(parts) == 2 && parts[1] == "result" && r.Method == http.MethodPost:
		taskID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid task id"})
			return
		}

		var req TaskResultRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		var task models.Task
		if err := db.DB.Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&task).Error; err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}

		status := req.Status
		if status != "success" && status != "failed" {
			status = "failed"
		}
		now := time.Now()
		db.DB.Model(&task).Updates(map[string]interface{}{
			"status":      status,
			"result":      req.Result,
			"finished_at": now,
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
