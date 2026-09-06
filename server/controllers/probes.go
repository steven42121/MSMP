// Package controllers 提供可用性探测 API。
package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/services"
)

// ProbesHandler GET /api/probes 列表，POST /api/probes 创建
func ProbesHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var probes []models.AvailProbe
		db.DB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&probes)
		writeJSON(w, http.StatusOK, probes)

	case http.MethodPost:
		var probe models.AvailProbe
		if err := json.NewDecoder(r.Body).Decode(&probe); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		probe.TenantID = tenantID
		probe.Enabled = true
		if probe.IntervalSec <= 0 {
			probe.IntervalSec = 60
		}
		if probe.TimeoutSec <= 0 {
			probe.TimeoutSec = 10
		}
		db.DB.Create(&probe)
		writeJSON(w, http.StatusCreated, probe)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// ProbeDetailHandler PUT/DELETE /api/probes/{id}
func ProbeDetailHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/probes/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var probe models.AvailProbe
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, getTenantID(r)).First(&probe).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "probe not found"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var updates map[string]interface{}
		json.NewDecoder(r.Body).Decode(&updates)
		db.DB.Model(&probe).Updates(updates)
		writeJSON(w, http.StatusOK, probe)

	case http.MethodDelete:
		db.DB.Delete(&probe)
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// ProbeRunHandler POST /api/probes/{id}/run 手动执行一次探测
func ProbeRunHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/probes/"):]
	idStr = idStr[len("/run"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var probe models.AvailProbe
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, getTenantID(r)).First(&probe).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "probe not found"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(probe.TimeoutSec)*time.Second)
	defer cancel()

	target := services.NormalizeProbeTarget(probe.Target, probe.Type)
	var result services.ProbeResult

	switch probe.Type {
	case "http", "https":
		result = services.ProbeHTTP(ctx, target, probe.TimeoutSec, probe.ExpectedCode)
	case "tcp":
		result = services.ProbeTCP(ctx, target, probe.TimeoutSec)
	case "ssl":
		result = services.ProbeSSL(ctx, target, probe.TimeoutSec)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported probe type: " + probe.Type})
		return
	}

	now := time.Now()
	db.DB.Model(&probe).Updates(map[string]interface{}{
		"last_status":   func() string { if result.Up { return "up" } else { return "down" } }(),
		"last_latency": result.Latency,
		"updated_at":   now,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"up":          result.Up,
		"latency_ms":  result.Latency,
		"error":       result.Error,
		"code":        result.Code,
		"ssl_issues":  result.Issues,
	})
}
