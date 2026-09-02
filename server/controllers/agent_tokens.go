package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
)

func generateAgentToken() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "msmp_" + hex.EncodeToString(b)
}

func AgentTokensHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var tokens []models.AgentToken
		query := db.DB.Where("tenant_id = ?", tenantID)
		if hostID := r.URL.Query().Get("host_id"); hostID != "" {
			query = query.Where("host_id = ?", hostID)
		}
		query.Order("created_at DESC").Find(&tokens)
		writeJSON(w, http.StatusOK, tokens)

	case http.MethodPost:
		var req struct {
			HostID      *uint  `json:"host_id"`
			Description string `json:"description"`
			ExpiresDays int    `json:"expires_days"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		token := models.AgentToken{
			TenantID:    tenantID,
			HostID:      req.HostID,
			Token:       generateAgentToken(),
			Description: req.Description,
		}
		if req.ExpiresDays > 0 {
			exp := time.Now().Add(time.Duration(req.ExpiresDays) * 24 * time.Hour)
			token.ExpiresAt = &exp
		}
		if err := db.DB.Create(&token).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create token"})
			return
		}
		writeJSON(w, http.StatusCreated, token)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func AgentTokenDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/agent-tokens/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token id required"})
		return
	}

	tokenID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid token id"})
		return
	}

	var token models.AgentToken
	if err := db.DB.Where("id = ? AND tenant_id = ?", tokenID, tenantID).First(&token).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "token not found"})
		return
	}

	switch r.Method {
	case http.MethodDelete:
		db.DB.Model(&token).Update("revoked", true)
		writeJSON(w, http.StatusOK, map[string]string{"message": "revoked"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
