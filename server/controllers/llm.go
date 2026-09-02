package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/services"
)

// LLMSettingsHandler GET/PUT /api/llm/settings
func LLMSettingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		keys := []string{"llm.base_url", "llm.api_key", "llm.model"}
		result := map[string]string{}
		for _, k := range keys {
			result[k] = GetSetting(k)
		}
		// api_key 返回脱敏版本
		if v, ok := result["llm.api_key"]; ok && v != "" {
			if len(v) > 8 {
				result["llm.api_key"] = v[:4] + "****" + v[len(v)-4:]
			} else {
				result["llm.api_key"] = "****"
			}
		}
		writeJSON(w, http.StatusOK, result)

	case http.MethodPut:
		var updates map[string]string
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		allowedKeys := map[string]bool{
			"llm.base_url": true,
			"llm.api_key":  true,
			"llm.model":    true,
		}
		for k, v := range updates {
			if !allowedKeys[k] {
				continue
			}
			var s models.Setting
			if err := db.DB.Where("key = ?", k).First(&s).Error; err != nil {
				db.DB.Create(&models.Setting{Key: k, Value: v})
			} else {
				db.DB.Model(&s).Update("value", v)
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "saved"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// AIChatHandler POST /api/ai/chat
func AIChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Query) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query required"})
		return
	}

	tenantID := getTenantID(r)
	result, err := services.ChatQuery(req.Query, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// AIAnalyzeHandler POST /api/ai/analyze-alert/:id
func AIAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/ai/analyze-alert/")
	var eventID uint
	if _, err := fmt.Sscanf(idStr, "%d", &eventID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}

	// 读取 host_id（从 URL 查询参数或直接从 event 查）
	hostIDStr := r.URL.Query().Get("host_id")
	var hostID uint
	if hostIDStr != "" {
		fmt.Sscanf(hostIDStr, "%d", &hostID)
	}

	if hostID == 0 {
		var event models.HostEvent
		if err := db.DB.First(&event, eventID).Error; err == nil {
			hostID = event.HostID
		}
	}

	if hostID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host_id required"})
		return
	}

	result, err := services.AnalyzeAlert(eventID, hostID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"event_id": eventID,
		"analysis": result,
	})
}

// AIGenerateReportHandler POST /api/ai/generate-report
func AIGenerateReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tenantID := getTenantID(r)
	result, err := services.GenerateReport(tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"report": result})
}
