// Package controllers 提供 Proxmox VE 管理 API。
package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"MSMP/server/collectors"
	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/services"
)

// requirePVEAdmin 校验 admin 角色 + pve 渠道存在。
func requirePVEAdmin(w http.ResponseWriter, r *http.Request, hostID, tenantID uint) bool {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可执行虚拟化操作"})
		return false
	}
	var count int64
	db.DB.Model(&models.ChannelBinding{}).
		Where("host_id = ? AND tenant_id = ? AND type = ? AND enabled = ?", hostID, tenantID, "pve", true).
		Count(&count)
	if count == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "主机未配置 Proxmox VE 渠道"})
		return false
	}
	return true
}

// connectPVE 解密渠道凭据并建立 PVE 客户端。
func connectPVE(r *http.Request, tenantID, hostID uint) (*services.PVEClient, *models.ChannelBinding, error) {
	if services.GlobalCredSvc == nil {
		return nil, nil, httpError("凭证服务未初始化")
	}

	var binding models.ChannelBinding
	if err := db.DB.Where("host_id = ? AND tenant_id = ? AND type = ? AND enabled = ?",
		hostID, tenantID, "pve", true).First(&binding).Error; err != nil {
		return nil, nil, httpError("主机未配置 Proxmox VE 渠道")
	}

	secret, err := services.GlobalCredSvc.Decrypt(binding.Credential)
	if err != nil {
		return nil, nil, httpError("凭证解密失败: " + err.Error())
	}

	username, password := collectors.ParsePVECredential(secret)
	client, err := services.NewPVEClient(r.Context(), binding.Address, username, password)
	if err != nil {
		return nil, nil, err
	}
	return client, &binding, nil
}

type httpErrorMessage string

func (e httpErrorMessage) Error() string { return string(e) }

func httpError(msg string) error { return httpErrorMessage(msg) }

// PVEGuestsHandler GET /api/hosts/{uuid}/pve/guests
func PVEGuestsHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if !requirePVEAdmin(w, r, host.ID, tenantID) {
		return
	}

	client, _, err := connectPVE(r, tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer client.Logout()

	guests, err := client.ListGuests(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   "pve_list_guests",
		Resource: "host:" + host.Hostname,
		Status:   200,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"guests": guests,
		"count":  len(guests),
	})
}

// PVEGuestPowerHandler POST /api/hosts/{uuid}/pve/guests/power
// body: {"node": "pve1", "vmid": 100, "vmtype": "qemu"|"lxc", "action": "start"|"stop"|...}
func PVEGuestPowerHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if !requirePVEAdmin(w, r, host.ID, tenantID) {
		return
	}

	var req struct {
		Node   string `json:"node"`
		VMID   int    `json:"vmid"`
		GuestType string `json:"vmtype"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求体解析失败"})
		return
	}
	if req.Node == "" || req.VMID <= 0 || req.GuestType == "" || req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node, vmid, vmtype, action 均必填"})
		return
	}

	client, _, err := connectPVE(r, tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer client.Logout()

	if err := client.PowerGuest(r.Context(), req.Node, req.GuestType, req.VMID, req.Action); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   "pve_power_guest",
		Resource: "guest:" + req.GuestType + ":" + req.Node + "/" + strconv.Itoa(req.VMID) + ":" + req.Action,
		Status:   200,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node":   req.Node,
		"vmid":   req.VMID,
		"type":   req.GuestType,
		"action": req.Action,
		"done":   true,
	})
}

// PVEStorageHandler GET /api/hosts/{uuid}/pve/storage
func PVEStorageHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if !requirePVEAdmin(w, r, host.ID, tenantID) {
		return
	}

	client, _, err := connectPVE(r, tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer client.Logout()

	storages, err := client.ListStorage(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"storages": storages,
		"count":    len(storages),
	})
}
