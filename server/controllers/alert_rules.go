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

func AlertRulesHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var rules []models.AlertRule
		query := db.DB.Where("tenant_id = ?", tenantID)
		if enabled := r.URL.Query().Get("enabled"); enabled != "" {
			query = query.Where("enabled = ?", enabled == "true")
		}
		query.Order("created_at DESC").Find(&rules)
		writeJSON(w, http.StatusOK, rules)

	case http.MethodPost:
		var rule models.AlertRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if rule.Name == "" || rule.Metric == "" || rule.Operator == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, metric, operator required"})
			return
		}
		rule.TenantID = tenantID
		if rule.Level == "" {
			rule.Level = "warning"
		}
		if err := db.DB.Create(&rule).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create rule"})
			return
		}
		writeJSON(w, http.StatusCreated, rule)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func AlertRuleDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/alert-rules/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rule id required"})
		return
	}

	ruleID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rule id"})
		return
	}

	var rule models.AlertRule
	if err := db.DB.Where("id = ? AND tenant_id = ?", ruleID, tenantID).First(&rule).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "rule not found"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var updates models.AlertRule
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		db.DB.Model(&rule).Updates(map[string]interface{}{
			"name":      updates.Name,
			"metric":    updates.Metric,
			"operator":  updates.Operator,
			"threshold": updates.Threshold,
			"level":     updates.Level,
			"enabled":   updates.Enabled,
		})
		db.DB.Where("id = ? AND tenant_id = ?", ruleID, tenantID).First(&rule)
		writeJSON(w, http.StatusOK, rule)
	case http.MethodDelete:
		db.DB.Delete(&rule)
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// AlertDetailHandler 告警确认/静音
func AlertDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/alerts/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "alert id required"})
		return
	}

	alertID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid alert id"})
		return
	}

	var event models.HostEvent
	if err := db.DB.Where("id = ? AND tenant_id = ?", alertID, tenantID).First(&event).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert not found"})
		return
	}

	switch {
	case len(parts) >= 2 && parts[1] == "ack" && r.Method == http.MethodPost:
		db.DB.Model(&event).Update("acknowledged", true)
		writeJSON(w, http.StatusOK, map[string]string{"message": "acknowledged"})
	case len(parts) >= 2 && parts[1] == "silence" && r.Method == http.MethodPost:
		var req struct {
			Minutes int `json:"minutes"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Minutes <= 0 {
			req.Minutes = 60
		}
		until := time.Now().Add(time.Duration(req.Minutes) * time.Minute)
		db.DB.Model(&event).Update("silenced_until", until)
		writeJSON(w, http.StatusOK, map[string]interface{}{"silenced_until": until})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
