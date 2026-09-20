package controllers

import (
	"net/http"

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

	p := newPagination(r)
	p.count(query, &models.AuditLog{})

	var logs []models.AuditLog
	p.scope(query.Order("created_at DESC")).Find(&logs)
	p.write(w, logs)
}

// auditLog 记录操作审计日志。
func auditLog(tenantID, userID uint, action, resource string, status int) {
	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   action,
		Resource: resource,
		Status:   status,
	})
}
