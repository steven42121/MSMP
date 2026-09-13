package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/plugins"
)

const maskedSecret = "******"

var pluginReg = plugins.NewRegistry()

// initPlugins 登记内置插件：通知插件 + 采集/探测元数据。
func initPlugins() {
	pluginReg.RegisterNotifier(plugins.WebhookPlugin{})
	pluginReg.RegisterNotifier(plugins.DingTalkPlugin{})
	pluginReg.RegisterNotifier(plugins.FeishuPlugin{})
	pluginReg.RegisterNotifier(plugins.WeComPlugin{})
	pluginReg.RegisterNotifier(plugins.SlackPlugin{})
	pluginReg.RegisterNotifier(plugins.SMTPPlugin{})

	collectorsMeta := []struct{ id, name, desc string }{
		{"collector-ssh", "SSH", "通过 SSH 远程命令采集"},
		{"collector-wac", "Windows Admin Center", "通过 WAC REST API 采集"},
		{"collector-baota", "宝塔面板", "通过宝塔面板 API 采集"},
		{"collector-1panel", "1Panel", "通过 1Panel API Key 采集"},
		{"collector-prometheus", "Prometheus / Node Exporter", "抓取 /metrics 采集"},
		{"collector-snmp", "SNMP", "通过 SNMP 协议采集网络设备"},
		{"collector-winrm", "WinRM", "通过 WinRM 采集 Windows"},
		{"collector-vsphere", "vSphere / ESXi", "通过 vSphere API 采集虚拟化"},
		{"collector-pve", "Proxmox VE", "通过 PVE API 采集虚拟化"},
	}
	for _, c := range collectorsMeta {
		pluginReg.RegisterMeta(plugins.PluginMeta{
			ID: c.id, Type: plugins.PluginTypeCollector, Name: c.name, Description: c.desc,
			ConfigFields: []plugins.ConfigField{
				{Key: "address", Label: "地址", Type: "string", Required: true},
			},
		})
	}

	probesMeta := []struct{ id, name, desc string }{
		{"probe-http", "HTTP 探测", "HTTP/HTTPS 可用性探测"},
		{"probe-tcp", "TCP 探测", "TCP 端口连通性探测"},
		{"probe-ssl", "SSL 证书探测", "SSL 证书有效期探测"},
	}
	for _, p := range probesMeta {
		pluginReg.RegisterMeta(plugins.PluginMeta{
			ID: p.id, Type: plugins.PluginTypeProbe, Name: p.name, Description: p.desc,
			ConfigFields: []plugins.ConfigField{
				{Key: "target", Label: "目标", Type: "string", Required: true},
			},
		})
	}
}

func init() {
	initPlugins()
}

func isSecretField(meta plugins.PluginMeta, key string) bool {
	for _, f := range meta.ConfigFields {
		if f.Key == key {
			return f.Secret
		}
	}
	return false
}

// encryptPluginConfig 对敏感字段加密后序列化存储。
func encryptPluginConfig(meta plugins.PluginMeta, cfg map[string]string) (string, error) {
	out := make(map[string]string, len(cfg))
	cred, err := credentialProvider()
	if err != nil {
		return "", err
	}
	for k, v := range cfg {
		if isSecretField(meta, k) && v != "" {
			enc, err := cred.Encrypt(v)
			if err != nil {
				return "", err
			}
			out[k] = enc
		} else {
			out[k] = v
		}
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

// decryptPluginConfig 解密实例配置中的敏感字段。
func decryptPluginConfig(meta plugins.PluginMeta, raw string) (map[string]string, error) {
	cfg := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	cred, err := credentialProvider()
	if err != nil {
		return nil, err
	}
	for k, v := range cfg {
		if isSecretField(meta, k) && v != "" {
			pt, err := cred.Decrypt(v)
			if err != nil {
				return nil, err
			}
			cfg[k] = pt
		}
	}
	return cfg, nil
}

// maskSecretConfig 掩码敏感字段用于响应。
func maskSecretConfig(meta plugins.PluginMeta, cfg map[string]string) map[string]string {
	out := make(map[string]string, len(cfg))
	for k, v := range cfg {
		if isSecretField(meta, k) && v != "" {
			out[k] = maskedSecret
		} else {
			out[k] = v
		}
	}
	return out
}

func instanceView(inst models.PluginInstance, meta plugins.PluginMeta) map[string]interface{} {
	view := map[string]interface{}{
		"id":         inst.ID,
		"plugin_id":  inst.PluginID,
		"type":       inst.Type,
		"name":       inst.Name,
		"enabled":    inst.Enabled,
		"config":     map[string]string{},
		"created_at": inst.CreatedAt,
		"updated_at": inst.UpdatedAt,
	}
	if cfg, err := decryptPluginConfig(meta, inst.Config); err == nil {
		view["config"] = maskSecretConfig(meta, cfg)
	}
	return view
}

// PluginsHandler GET /api/plugins
func PluginsHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, pluginReg.List())
}

// PluginInstancesHandler GET/POST /api/plugins/instances
func PluginInstancesHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var instances []models.PluginInstance
		db.DB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&instances)
		views := make([]map[string]interface{}, 0, len(instances))
		for _, inst := range instances {
			meta, _ := pluginReg.Meta(inst.PluginID)
			views = append(views, instanceView(inst, meta))
		}
		writeJSON(w, http.StatusOK, views)

	case http.MethodPost:
		var req struct {
			PluginID string            `json:"plugin_id"`
			Name     string            `json:"name"`
			Config   map[string]string `json:"config"`
			Enabled  *bool             `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		meta, ok := pluginReg.Meta(req.PluginID)
		if !ok || meta.Type != plugins.PluginTypeNotifier {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown notifier plugin"})
			return
		}
		if err := validatePluginConfig(meta, req.Config); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		enc, err := encryptPluginConfig(meta, req.Config)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential service unavailable: " + err.Error()})
			return
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		inst := models.PluginInstance{
			TenantID: tenantID,
			PluginID: req.PluginID,
			Type:     string(plugins.PluginTypeNotifier),
			Name:     req.Name,
			Config:   enc,
			Enabled:  enabled,
		}
		if inst.Name == "" {
			inst.Name = meta.Name
		}
		if err := db.DB.Create(&inst).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create instance"})
			return
		}
		writeJSON(w, http.StatusCreated, instanceView(inst, meta))

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func validatePluginConfig(meta plugins.PluginMeta, cfg map[string]string) error {
	for _, f := range meta.ConfigFields {
		if f.Required && strings.TrimSpace(cfg[f.Key]) == "" {
			return fmt.Errorf("missing required field: %s", f.Key)
		}
	}
	return nil
}

// PluginInstanceDetailHandler PUT/DELETE /api/plugins/instances/{id}
func PluginInstanceDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	idStr := strings.TrimPrefix(r.URL.Path, "/api/plugins/instances/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var inst models.PluginInstance
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&inst).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "instance not found"})
		return
	}
	meta, _ := pluginReg.Meta(inst.PluginID)

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name    string            `json:"name"`
			Config  map[string]string `json:"config"`
			Enabled *bool             `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		updates := map[string]interface{}{}
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if req.Config != nil {
			if err := validatePluginConfig(meta, req.Config); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			merged, err := mergeConfig(meta, inst.Config, req.Config)
			if err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential service unavailable: " + err.Error()})
				return
			}
			updates["config"] = merged
		}
		if err := db.DB.Model(&inst).Updates(updates).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update instance"})
			return
		}
		db.DB.Where("id = ?", inst.ID).First(&inst)
		writeJSON(w, http.StatusOK, instanceView(inst, meta))

	case http.MethodDelete:
		if err := db.DB.Delete(&inst).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete instance"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"deleted": "true"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// mergeConfig 合并更新配置：敏感字段若传入掩码值则保留原加密值，否则重新加密。
func mergeConfig(meta plugins.PluginMeta, originalRaw string, newCfg map[string]string) (string, error) {
	orig, err := decryptPluginConfig(meta, originalRaw)
	if err != nil {
		return "", err
	}
	for k, v := range newCfg {
		if isSecretField(meta, k) && v == maskedSecret {
			continue
		}
		orig[k] = v
	}
	return encryptPluginConfig(meta, orig)
}

// PluginInstanceTestHandler POST /api/plugins/instances/{id}/test
func PluginInstanceTestHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	idStr := strings.TrimPrefix(r.URL.Path, "/api/plugins/instances/")
	idStr = strings.TrimSuffix(idStr, "/test")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var inst models.PluginInstance
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&inst).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "instance not found"})
		return
	}
	meta, _ := pluginReg.Meta(inst.PluginID)
	p, ok := pluginReg.GetNotifier(inst.PluginID)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown notifier plugin"})
		return
	}
	cfg, err := decryptPluginConfig(meta, inst.Config)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential service unavailable: " + err.Error()})
		return
	}
	if err := p.Test(r.Context(), cfg); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// DispatchNotifiers 将通知分发到租户全部启用的通知实例，返回是否有实例成功发送。
// 由告警引擎调用（无 HTTP request 上下文）。
func DispatchNotifiers(tenantID uint, n plugins.Notification) bool {
	var instances []models.PluginInstance
	db.DB.Where("tenant_id = ? AND enabled = ? AND type = ?", tenantID, true, "notifier").Find(&instances)

	sent := false
	for _, inst := range instances {
		p, ok := pluginReg.GetNotifier(inst.PluginID)
		if !ok {
			continue
		}
		meta, _ := pluginReg.Meta(inst.PluginID)
		cfg, err := decryptPluginConfig(meta, inst.Config)
		if err != nil {
			log.Printf("notify plugin %s decrypt failed: %v", inst.PluginID, err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		if err := p.Send(ctx, cfg, n); err != nil {
			cancel()
			log.Printf("notify plugin %s failed: %v", inst.PluginID, err)
			continue
		}
		cancel()
		sent = true
	}
	return sent
}