// Package controllers contains system maintenance API handlers.
package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"MSMP/server/db"
	"MSMP/server/models"
)

// FlushCachesRequest 清理缓存请求
type FlushCachesRequest struct {
	HostUUID   string `json:"host_uuid"`
	CacheType  string `json:"cache_type"` // pages, dentries, inodes, all
}

// FlushCachesHandler 一键清理内存缓存
func FlushCachesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tenantID := getTenantID(r)

	var req FlushCachesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.HostUUID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host_uuid required"})
		return
	}

	// 验证缓存类型
	cacheType := req.CacheType
	if cacheType == "" {
		cacheType = "all"
	}
	validTypes := map[string]bool{"pages": true, "dentries": true, "inodes": true, "all": true}
	if !validTypes[cacheType] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid cache_type"})
		return
	}

	// 查找主机
	var host models.Host
	if err := db.DB.Where("uuid = ? AND tenant_id = ?", req.HostUUID, tenantID).First(&host).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
		return
	}

	// 构建清理命令
	var cmd string
	switch cacheType {
	case "pages":
		cmd = "sync && echo 1 > /proc/sys/vm/drop_caches"
	case "dentries":
		cmd = "sync && echo 2 > /proc/sys/vm/drop_caches"
	case "inodes":
		cmd = "sync && echo 4 > /proc/sys/vm/drop_caches"
	default: // all
		cmd = "sync && echo 3 > /proc/sys/vm/drop_caches"
	}

	// 创建任务
	task := models.Task{
		TenantID:  tenantID,
		HostID:    host.ID,
		Type:      "flush_caches",
		Command:   cmd,
		Status:    "pending",
		CreatedBy: getUserID(r),
	}
	if err := db.DB.Create(&task).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create task"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"task_id":   task.ID,
		"host":      host.Hostname,
		"cache_type": cacheType,
		"message":   fmt.Sprintf("已提交清理缓存任务，类型: %s", cacheType),
	})
}

// DownsampleHandler 手动触发一次时序数据降采样与清理。
func DownsampleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	runDownsampleCleanup()
	writeJSON(w, http.StatusOK, map[string]string{"message": "降采样清理已触发"})
}
