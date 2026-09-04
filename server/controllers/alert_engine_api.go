// Package controllers contains alert engineering API handlers.
package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"MSMP/server/db"
	"MSMP/server/models"
)

// AlertSuppressionsHandler GET/POST /api/alert-suppressions
func AlertSuppressionsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var rules []models.AlertSuppression
		db.DB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&rules)
		writeJSON(w, http.StatusOK, rules)

	case http.MethodPost:
		var rule models.AlertSuppression
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		rule.TenantID = tenantID
		if rule.WindowMinutes <= 0 {
			rule.WindowMinutes = 30
		}
		db.DB.Create(&rule)
		writeJSON(w, http.StatusCreated, rule)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// AlertSuppressionDetailHandler PUT/DELETE /api/alert-suppressions/:id
func AlertSuppressionDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/alert-suppressions/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var rule models.AlertSuppression
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&rule).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var updates models.AlertSuppression
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		db.DB.Model(&rule).Updates(map[string]interface{}{
			"name":            updates.Name,
			"host_id":         updates.HostID,
			"metric":          updates.Metric,
			"level":           updates.Level,
			"window_minutes":  updates.WindowMinutes,
			"enabled":         updates.Enabled,
		})
		writeJSON(w, http.StatusOK, rule)

	case http.MethodDelete:
		db.DB.Delete(&rule)
		writeJSON(w, http.StatusOK, map[string]string{"deleted": "true"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// AlertSilencesHandler GET/POST /api/alert-silences
func AlertSilencesHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var rules []models.AlertSilence
		db.DB.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
			Order("created_at DESC").Find(&rules)
		writeJSON(w, http.StatusOK, rules)

	case http.MethodPost:
		var rule models.AlertSilence
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		rule.TenantID = tenantID
		rule.CreatorID = getUserID(r)
		db.DB.Create(&rule)
		writeJSON(w, http.StatusCreated, rule)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// AlertSilenceDetailHandler PUT/DELETE /api/alert-silences/:id
func AlertSilenceDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/alert-silences/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var rule models.AlertSilence
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&rule).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var updates models.AlertSilence
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		db.DB.Model(&rule).Updates(map[string]interface{}{
			"name":       updates.Name,
			"host_id":    updates.HostID,
			"label_key":  updates.LabelKey,
			"label_value": updates.LabelValue,
			"level":      updates.Level,
			"start_at":   updates.StartAt,
			"end_at":     updates.EndAt,
		})
		writeJSON(w, http.StatusOK, rule)

	case http.MethodDelete:
		db.DB.Delete(&rule)
		writeJSON(w, http.StatusOK, map[string]string{"deleted": "true"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// AlertEscalationsHandler GET/POST /api/alert-escalations
func AlertEscalationsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var rules []models.AlertEscalation
		db.DB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&rules)
		writeJSON(w, http.StatusOK, rules)

	case http.MethodPost:
		var rule models.AlertEscalation
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		rule.TenantID = tenantID
		if rule.NotifyAfterMin <= 0 {
			rule.NotifyAfterMin = 60
		}
		if rule.RetryCount <= 0 {
			rule.RetryCount = 3
		}
		db.DB.Create(&rule)
		writeJSON(w, http.StatusCreated, rule)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// AlertEscalationDetailHandler PUT/DELETE /api/alert-escalations/:id
func AlertEscalationDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/alert-escalations/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var rule models.AlertEscalation
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&rule).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var updates models.AlertEscalation
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		db.DB.Model(&rule).Updates(map[string]interface{}{
			"name":              updates.Name,
			"trigger_level":     updates.TriggerLevel,
			"notify_after_min":  updates.NotifyAfterMin,
			"retry_count":       updates.RetryCount,
			"enabled":           updates.Enabled,
		})
		writeJSON(w, http.StatusOK, rule)

	case http.MethodDelete:
		db.DB.Delete(&rule)
		writeJSON(w, http.StatusOK, map[string]string{"deleted": "true"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
