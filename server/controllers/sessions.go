// Package controllers 提供 WebSSH 会话录制 API。
package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
)

// SessionRecordingsHandler GET /api/sessions 列出录制文件
func SessionRecordingsHandler(w http.ResponseWriter, r *http.Request) {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可访问会话录制"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		recorder := getRecorder()
		infos, err := recorder.ListRecordings()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, infos)

	case http.MethodDelete:
		// DELETE /api/sessions/{filename}
		filename := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		if filename == "" || strings.Contains(filename, "/") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "非法文件名"})
			return
		}
		dataDir := filepath.Join(".", "data", "sessions")
		if err := os.Remove(filepath.Join(dataDir, filename)); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "文件不存在"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "已删除"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// SessionDownloadHandler GET /api/sessions/{filename} 下载录制内容
func SessionDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可访问会话录制"})
		return
	}

	filename := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if filename == "" || strings.Contains(filename, "/") || !strings.HasSuffix(filename, ".log") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "非法文件名"})
		return
	}

	dataDir := filepath.Join(".", "data", "sessions")
	path := filepath.Join(dataDir, filename)
	absPath, err := filepath.Abs(path)
	if err != nil || !strings.HasPrefix(absPath, filepath.Clean(dataDir)+string(filepath.Separator)) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "非法路径"})
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "文件不存在"})
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write(content)
}

// SessionAuditHandler GET /api/sessions/audit 列出审计日志中的会话记录
func SessionAuditHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var logs []models.AuditLog
	db.DB.Where("tenant_id = ? AND action = 'webssh_connect' OR action = 'webssh_disconnect'", tenantID).
		Order("created_at DESC").Limit(limit).Find(&logs)

	type sessionRecord struct {
		ID         uint      `json:"id"`
		UserID     uint      `json:"user_id"`
		Username   string    `json:"username"`
		Resource   string    `json:"resource"`
		Action     string    `json:"action"`
		CreatedAt  time.Time `json:"created_at"`
	}
	// 简化：直接返回 audit logs
	writeJSON(w, http.StatusOK, logs)
}
