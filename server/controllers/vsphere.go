// Package controllers 提供 vSphere 虚拟化管理 API。
package controllers

import (
	"encoding/json"
	"net/http"

	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/services"
)

// requireVSphereAdmin 校验 admin 角色 + vsphere 渠道存在。
func requireVSphereAdmin(w http.ResponseWriter, r *http.Request, hostID, tenantID uint) bool {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可执行虚拟化操作"})
		return false
	}
	var count int64
	db.DB.Model(&models.ChannelBinding{}).
		Where("host_id = ? AND tenant_id = ? AND type = ? AND enabled = ?", hostID, tenantID, "vsphere", true).
		Count(&count)
	if count == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "主机未配置 vSphere 渠道"})
		return false
	}
	return true
}

// VSphereVMsHandler GET /api/hosts/{uuid}/vsphere/vms
func VSphereVMsHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if !requireVSphereAdmin(w, r, host.ID, tenantID) {
		return
	}

	mgr, err := services.ConnectVSphere(r.Context(), tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer mgr.Close(r.Context())

	vms, err := mgr.ListVMs(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   "vsphere_list_vms",
		Resource: "host:" + host.Hostname,
		Status:   200,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"vms":   vms,
		"count": len(vms),
	})
}

// VSphereVMPowerHandler POST /api/hosts/{uuid}/vsphere/vms/{name}/power
// body: {"action": "on"|"off"|"reset"|"suspend"}
func VSphereVMPowerHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint, vmName string) {
	if !requireVSphereAdmin(w, r, host.ID, tenantID) {
		return
	}

	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action 必填（on/off/reset/suspend）"})
		return
	}
	if !map[string]bool{"on": true, "off": true, "reset": true, "suspend": true}[req.Action] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的 action: " + req.Action})
		return
	}

	mgr, err := services.ConnectVSphere(r.Context(), tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer mgr.Close(r.Context())

	if err := mgr.PowerVM(r.Context(), vmName, req.Action); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   "vsphere_power_vm",
		Resource: "vm:" + vmName + ":" + req.Action,
		Status:   200,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"vm":     vmName,
		"action": req.Action,
		"done":   true,
	})
}

// VSphereDatastoresHandler GET /api/hosts/{uuid}/vsphere/datastores
func VSphereDatastoresHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if !requireVSphereAdmin(w, r, host.ID, tenantID) {
		return
	}

	mgr, err := services.ConnectVSphere(r.Context(), tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer mgr.Close(r.Context())

	dss, err := mgr.ListDatastores(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"datastores": dss,
		"count":      len(dss),
	})
}