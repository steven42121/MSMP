package controllers

import (
	"net/http"
	"strconv"

	"MSMP/server/db"
	"MSMP/server/models"
)

func AuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tenantID := getTenantID(r)
	query := db.DB.Where("tenant_id = ?", tenantID)

	if action := r.URL.Query().Get("action"); action != "" {
		query = query.Where("action = ?", action)
	}

	var total int64
	query.Model(&models.AuditLog{}).Count(&total)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var logs []models.AuditLog
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": logs, "total": total, "page": page, "page_size": pageSize,
	})
}
