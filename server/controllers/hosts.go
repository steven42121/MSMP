package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"MSMP/server/db"
	"MSMP/server/models"
)

type HostCreateRequest struct {
	Hostname  string `json:"hostname"`
	IP        string `json:"ip"`
	OS        string `json:"os"`
	OSVersion string `json:"os_version"`
	Arch      string `json:"arch"`
}

func generateToken() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "msmp_" + hex.EncodeToString(b)
}

// HostsHandler 主机列表 / 批量操作 / 添加
func HostsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		// 查询主机列表
		var hosts []models.Host
		query := db.DB.Where("tenant_id = ?", tenantID)

		// 筛选条件
		if status := r.URL.Query().Get("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if os := r.URL.Query().Get("os"); os != "" {
			query = query.Where("os = ?", os)
		}
		if keyword := r.URL.Query().Get("keyword"); keyword != "" {
			query = query.Where("hostname LIKE ? OR ip LIKE ?",
				"%"+keyword+"%", "%"+keyword+"%")
		}

		// 分页
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 || pageSize > 100 {
			pageSize = 20
		}

		var total int64
		query.Model(&models.Host{}).Count(&total)
		query.Order("last_heartbeat DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&hosts)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":      hosts,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})

	case http.MethodPost:
		// 手动添加主机
		var req HostCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if req.Hostname == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname required"})
			return
		}

		uuid := generateToken()
		host := models.Host{
			TenantID: tenantID,
			UUID:     uuid,
			Hostname: req.Hostname,
			IP:       req.IP,
			OS:       req.OS,
			OSVersion: req.OSVersion,
			Arch:     req.Arch,
			Status:   "pending",
		}
		if err := db.DB.Create(&host).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create host"})
			return
		}

		token := models.AgentToken{
			TenantID:    tenantID,
			HostID:      &host.ID,
			Token:       generateToken(),
			Description: "manual created for " + req.Hostname,
		}
		db.DB.Create(&token)

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"host":        host,
			"agent_token": token.Token,
		})

	case http.MethodDelete:
		// 批量删除主机
		var req struct {
			IDs []uint `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		db.DB.Where("tenant_id = ? AND id IN ?", tenantID, req.IDs).Delete(&models.Host{})
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// HostDetailHandler 单个主机详情操作
func HostDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	// 解析路径 /api/hosts/{uuid}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/hosts/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host uuid required"})
		return
	}
	uuid := parts[0]
	subResource := ""
	if len(parts) > 1 {
		subResource = parts[1]
	}
	subAction := ""
	if len(parts) > 2 {
		subAction = parts[2]
	}

	var host models.Host
	if err := db.DB.Where("uuid = ? AND tenant_id = ?", uuid, tenantID).First(&host).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
		return
	}

	switch {
	case subResource == "tags" && r.Method == http.MethodGet:
		var tags []models.HostTag
		db.DB.Where("host_id = ?", host.ID).Find(&tags)
		writeJSON(w, http.StatusOK, tags)

	case subResource == "tags" && r.Method == http.MethodPost:
		var tag models.HostTag
		json.NewDecoder(r.Body).Decode(&tag)
		tag.HostID = host.ID
		tag.TenantID = tenantID
		db.DB.Create(&tag)
		writeJSON(w, http.StatusCreated, tag)

	case subResource == "tags" && r.Method == http.MethodDelete:
		tagID := ""
		if len(parts) > 2 {
			tagID = parts[2]
		}
		db.DB.Where("id = ? AND host_id = ?", tagID, host.ID).Delete(&models.HostTag{})
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})

	case subResource == "metrics" && r.Method == http.MethodGet:
		// 查询监控数据
		var metrics []models.MetricSample
		limit := 60
		if l := r.URL.Query().Get("limit"); l != "" {
			limit, _ = strconv.Atoi(l)
		}
		db.DB.Where("host_id = ?", host.ID).
			Order("timestamp DESC").Limit(limit).Find(&metrics)
		writeJSON(w, http.StatusOK, metrics)

	case subResource == "events" && r.Method == http.MethodGet:
		var events []models.HostEvent
		db.DB.Where("host_id = ?", host.ID).
			Order("created_at DESC").Limit(100).Find(&events)
		writeJSON(w, http.StatusOK, events)

	case subResource == "assets" && r.Method == http.MethodGet:
		var snapshots []models.AssetSnapshot
		db.DB.Where("host_id = ?", host.ID).
			Order("created_at DESC").Limit(10).Find(&snapshots)
		writeJSON(w, http.StatusOK, snapshots)

	case subResource == "channels" && r.Method == http.MethodGet:
		ChannelsListHandler(w, r, host, tenantID)
	case subResource == "channels" && r.Method == http.MethodPost:
		ChannelsCreateHandler(w, r, host, tenantID, getUserID(r))

	case subResource == "ssh" && r.Method == http.MethodGet:
		WebSSHHandler(w, r, &host, tenantID, getUserID(r))

	case subResource == "files" && r.Method == http.MethodGet && subAction == "download":
		FileDownloadHandler(w, r, &host, tenantID, getUserID(r))
	case subResource == "files" && r.Method == http.MethodGet:
		FileListHandler(w, r, &host, tenantID, getUserID(r))
	case subResource == "files" && r.Method == http.MethodPost && subAction == "upload":
		FileUploadHandler(w, r, &host, tenantID, getUserID(r))
	case subResource == "files" && r.Method == http.MethodPost && subAction == "mkdir":
		FileMkdirHandler(w, r, &host, tenantID, getUserID(r))
	case subResource == "files" && r.Method == http.MethodPost && subAction == "rename":
		FileRenameHandler(w, r, &host, tenantID, getUserID(r))
	case subResource == "files" && r.Method == http.MethodDelete:
		FileDeleteHandler(w, r, &host, tenantID, getUserID(r))

	case subResource == "vsphere" && subAction == "vms" && r.Method == http.MethodGet:
		VSphereVMsHandler(w, r, &host, tenantID, getUserID(r))
	case subResource == "vsphere" && subAction == "vms" && len(parts) >= 6 && parts[5] == "power" && r.Method == http.MethodPost:
		VSphereVMPowerHandler(w, r, &host, tenantID, getUserID(r), parts[4])
	case subResource == "vsphere" && subAction == "datastores" && r.Method == http.MethodGet:
		VSphereDatastoresHandler(w, r, &host, tenantID, getUserID(r))

	case subResource == "pve" && subAction == "guests" && r.Method == http.MethodGet:
		PVEGuestsHandler(w, r, &host, tenantID, getUserID(r))
	case subResource == "pve" && subAction == "guests" && len(parts) >= 4 && parts[3] == "power" && r.Method == http.MethodPost:
		PVEGuestPowerHandler(w, r, &host, tenantID, getUserID(r))
	case subResource == "pve" && subAction == "storage" && r.Method == http.MethodGet:
		PVEStorageHandler(w, r, &host, tenantID, getUserID(r))

	case subResource == "agent" && subAction == "upgrade" && r.Method == http.MethodPost:
		AgentUpgradeHandler(w, r)

	case r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, host)

	case r.Method == http.MethodPut:
		var updates map[string]interface{}
		json.NewDecoder(r.Body).Decode(&updates)
		// 只允许更新部分字段
		allowedFields := map[string]bool{"hostname": true, "os_version": true}
		filtered := make(map[string]interface{})
		for k, v := range updates {
			if allowedFields[k] {
				filtered[k] = v
			}
		}
		if len(filtered) > 0 {
			db.DB.Model(&host).Updates(filtered)
		}
		writeJSON(w, http.StatusOK, host)

	case r.Method == http.MethodDelete:
		db.DB.Delete(&host)
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}