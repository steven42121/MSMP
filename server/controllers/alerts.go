package controllers

import (
	"net/http"
	"strconv"

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
}