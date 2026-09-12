// Package controllers 提供 vSphere 虚拟化管理 API。
package controllers

import (
	"encoding/json"
	"net/http"

	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/services"
)

// vsphereManager 校验 admin 角色、渠道存在并建立连接；失败时已写响应，返回 false。
func vsphereManager(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID uint) (*services.VSphereManager, bool) {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可执行虚拟化操作"})
		return nil, false
	}
	var count int64
	db.DB.Model(&models.ChannelBinding{}).
		Where("host_id = ? AND tenant_id = ? AND type = ? AND enabled = ?", host.ID, tenantID, "vsphere", true).
		Count(&count)
	if count == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "主机未配置 vSphere 渠道"})
		return nil, false
	}
	mgr, err := services.ConnectVSphere(r.Context(), tenantID, host.ID)
	if err != nil {
		writeUpstreamErr(w, err)
		return nil, false
	}
	return mgr, true
}

// writeUpstreamErr 统一上游服务（vSphere/PVE 等）错误响应。
func writeUpstreamErr(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
}

// VSphereVMsHandler GET /api/hosts/{uuid}/vsphere/vms
func VSphereVMsHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	mgr, ok := vsphereManager(w, r, host, tenantID)
	if !ok {
		return
	}
	defer mgr.Close(r.Context())

	vms, err := mgr.ListVMs(r.Context())
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}

	auditLog(tenantID, userID, "vsphere_list_vms", "host:"+host.Hostname, 200)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"vms":   vms,
		"count": len(vms),
	})
}

// VSphereVMPowerHandler POST /api/hosts/{uuid}/vsphere/vms/{name}/power
// body: {"action": "on"|"off"|"reset"|"suspend"}
func VSphereVMPowerHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint, vmName string) {
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

	mgr, ok := vsphereManager(w, r, host, tenantID)
	if !ok {
		return
	}
	defer mgr.Close(r.Context())

	if err := mgr.PowerVM(r.Context(), vmName, req.Action); err != nil {
		writeUpstreamErr(w, err)
		return
	}

	auditLog(tenantID, userID, "vsphere_power_vm", "vm:"+vmName+":"+req.Action, 200)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"vm":     vmName,
		"action": req.Action,
		"done":   true,
	})
}

// VSphereDatastoresHandler GET /api/hosts/{uuid}/vsphere/datastores
func VSphereDatastoresHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	mgr, ok := vsphereManager(w, r, host, tenantID)
	if !ok {
		return
	}
	defer mgr.Close(r.Context())

	dss, err := mgr.ListDatastores(r.Context())
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"datastores": dss,
		"count":      len(dss),
	})
}

// VSphereNetworksHandler GET /api/hosts/{uuid}/vsphere/networks
func VSphereNetworksHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	mgr, ok := vsphereManager(w, r, host, tenantID)
	if !ok {
		return
	}
	defer mgr.Close(r.Context())

	nets, err := mgr.ListNetworks(r.Context())
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"networks": nets,
		"count":    len(nets),
	})
}

// VSphereSnapshotsHandler GET /api/hosts/{uuid}/vsphere/vms/{name}/snapshots
func VSphereSnapshotsHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint, vmName string) {
	mgr, ok := vsphereManager(w, r, host, tenantID)
	if !ok {
		return
	}
	defer mgr.Close(r.Context())

	snaps, err := mgr.ListVMSnapshots(r.Context(), vmName)
	if err != nil {
		writeUpstreamErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"vm":        vmName,
		"snapshots": snaps,
		"count":     len(snaps),
	})
}

// VSphereSnapshotCreateHandler POST /api/hosts/{uuid}/vsphere/vms/{name}/snapshots
// body: {"name": "...", "description": "..."}
func VSphereSnapshotCreateHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint, vmName string) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name 必填"})
		return
	}

	mgr, ok := vsphereManager(w, r, host, tenantID)
	if !ok {
		return
	}
	defer mgr.Close(r.Context())

	if err := mgr.CreateVMSnapshot(r.Context(), vmName, req.Name, req.Description); err != nil {
		writeUpstreamErr(w, err)
		return
	}
	auditLog(tenantID, userID, "vsphere_create_snapshot", "vm:"+vmName+":snapshot:"+req.Name, 200)
	writeJSON(w, http.StatusOK, map[string]interface{}{"created": true, "vm": vmName, "name": req.Name})
}

// VSphereSnapshotDeleteHandler DELETE /api/hosts/{uuid}/vsphere/vms/{name}/snapshots
// query: snap_name
func VSphereSnapshotDeleteHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint, vmName string) {
	snapName := r.URL.Query().Get("snap_name")
	if snapName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "snap_name 必填"})
		return
	}

	mgr, ok := vsphereManager(w, r, host, tenantID)
	if !ok {
		return
	}
	defer mgr.Close(r.Context())

	if err := mgr.DeleteVMSnapshot(r.Context(), vmName, snapName); err != nil {
		writeUpstreamErr(w, err)
		return
	}
	auditLog(tenantID, userID, "vsphere_delete_snapshot", "vm:"+vmName+":snapshot:"+snapName, 200)
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "vm": vmName, "name": snapName})
}

// VSphereSnapshotRollbackHandler POST /api/hosts/{uuid}/vsphere/vms/{name}/snapshots/rollback
func VSphereSnapshotRollbackHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint, vmName string) {
	var req struct {
		SnapName string `json:"snap_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SnapName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "snap_name 必填"})
		return
	}

	mgr, ok := vsphereManager(w, r, host, tenantID)
	if !ok {
		return
	}
	defer mgr.Close(r.Context())

	if err := mgr.RevertVMSnapshot(r.Context(), vmName, req.SnapName); err != nil {
		writeUpstreamErr(w, err)
		return
	}
	auditLog(tenantID, userID, "vsphere_rollback_snapshot", "vm:"+vmName+":snapshot:"+req.SnapName, 200)
	writeJSON(w, http.StatusOK, map[string]interface{}{"rolled_back": true, "vm": vmName, "name": req.SnapName})
}
