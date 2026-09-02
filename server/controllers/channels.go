package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"MSMP/server/collectors"
	"MSMP/server/config"
	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/services"
)

var (
	credService   *services.CredentialService
	credOnce      sync.Once
	credInitErr   error
	channelReg    = collectors.NewRegistry()
)

func initCollectors() {
	credOnce.Do(func() {
		channelReg.Register(&collectors.SSHChannel{})
		channelReg.Register(&collectors.WACChannel{})
		channelReg.Register(&collectors.BaoTaChannel{})
		channelReg.Register(&collectors.PrometheusChannel{})
		channelReg.Register(&collectors.SNMPChannel{})
		channelReg.Register(&collectors.WinRMChannel{})
		if config.C != nil && config.C.Security.CredentialKey != "" {
			credService, credInitErr = services.NewCredentialService(config.C)
		}
	})
}

func credentialProvider() (services.CredentialService, error) {
	initCollectors()
	if credInitErr != nil {
		return services.CredentialService{}, credInitErr
	}
	if credService == nil {
		return services.CredentialService{}, fmt.Errorf("credential key not configured")
	}
	return *credService, nil
}

// decryptor adapts services.CredentialService to collectors.CredentialProvider.
type decryptor struct{ s services.CredentialService }

func (d decryptor) Decrypt(ciphertext string) (string, error) { return d.s.Decrypt(ciphertext) }

// ChannelsListHandler GET /api/hosts/{uuid}/channels
func ChannelsListHandler(w http.ResponseWriter, r *http.Request, host models.Host, tenantID uint) {
	var bindings []models.ChannelBinding
	db.DB.Where("host_id = ? AND tenant_id = ?", host.ID, tenantID).
		Order("priority asc").Find(&bindings)
	writeJSON(w, http.StatusOK, bindings)
}

// ChannelsCreateHandler POST /api/hosts/{uuid}/channels
func ChannelsCreateHandler(w http.ResponseWriter, r *http.Request, host models.Host, tenantID, userID uint) {
	var req struct {
		Type     string `json:"type"`
		Address  string `json:"address"`
		AuthMode string `json:"auth_mode"`
		Username string `json:"username"`
		Secret   string `json:"secret"`
		Priority int    `json:"priority"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.Type == "" || req.Address == "" || req.AuthMode == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type, address, auth_mode are required"})
		return
	}
	if req.Type != "ssh" && req.Type != "wac" && req.Type != "baota" && req.Type != "prometheus" && req.Type != "snmp" && req.Type != "winrm" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported channel type"})
		return
	}

	cred, err := credentialProvider()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential service unavailable: " + err.Error()})
		return
	}
	enc, err := cred.Encrypt(req.Secret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encrypt credential"})
		return
	}

	// 防止同类型重复启用
	var existing models.ChannelBinding
	if err := db.DB.Where("host_id = ? AND tenant_id = ? AND type = ? AND enabled = ?",
		host.ID, tenantID, req.Type, true).First(&existing).Error; err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "an enabled channel of this type already exists"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.Priority == 0 {
		req.Priority = 100
	}
	binding := models.ChannelBinding{
		TenantID:   tenantID,
		HostID:     host.ID,
		Type:       req.Type,
		Address:    req.Address,
		AuthMode:   req.AuthMode,
		Username:   req.Username,
		Credential: enc,
		Priority:   req.Priority,
		Enabled:    enabled,
	}
	if err := db.DB.Create(&binding).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create channel"})
		return
	}

	// 审计日志
	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   "create",
		Resource: "channel " + req.Type + " host:" + host.UUID,
		Method:   http.MethodPost,
		Status:   http.StatusCreated,
	})

	// 创建后自动探测
	go probeChannel(binding, cred)

	writeJSON(w, http.StatusCreated, binding)
}

// ChannelDetailHandler PUT/DELETE /api/channels/{id}
func ChannelDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	userID := getUserID(r)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/channels/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "channel id required"})
		return
	}
	idStr := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid channel id"})
		return
	}

	var binding models.ChannelBinding
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&binding).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "channel not found"})
		return
	}

	switch {
	case sub == "probe" && r.Method == http.MethodPost:
		cred, err := credentialProvider()
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		res := runProbe(ctx, &binding, cred)
		writeJSON(w, http.StatusOK, res)

	case r.Method == http.MethodPut:
		var req struct {
			Priority *int   `json:"priority"`
			Enabled  *bool  `json:"enabled"`
			Address  *string `json:"address"`
			Username *string `json:"username"`
			Secret   *string `json:"secret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		updates := map[string]interface{}{}
		if req.Priority != nil {
			updates["priority"] = *req.Priority
		}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if req.Address != nil {
			updates["address"] = *req.Address
		}
		if req.Username != nil {
			updates["username"] = *req.Username
		}
		if req.Secret != nil {
			cred, err := credentialProvider()
			if err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
				return
			}
			enc, err := cred.Encrypt(*req.Secret)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encrypt failed"})
				return
			}
			updates["credential"] = enc
		}
		if len(updates) > 0 {
			db.DB.Model(&binding).Updates(updates)
		}
		db.DB.Create(&models.AuditLog{
			TenantID: tenantID, UserID: userID, Action: "update",
			Resource: "channel " + binding.Type, Method: http.MethodPut, Status: http.StatusOK,
		})
		db.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&binding)
		writeJSON(w, http.StatusOK, binding)

	case r.Method == http.MethodDelete:
		credID := binding.Credential
		db.DB.Delete(&binding)
		_ = credID
		db.DB.Create(&models.AuditLog{
			TenantID: tenantID, UserID: userID, Action: "delete",
			Resource: "channel " + binding.Type, Method: http.MethodDelete, Status: http.StatusOK,
		})
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// SSHKeypairHandler POST /api/channels/ssh-keypair
func SSHKeypairHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	tenantID := getTenantID(r)
	hostUUID := r.URL.Query().Get("host_uuid")
	cred, err := credentialProvider()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	pub, priv, err := services.GenerateSSHKeypair()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "keygen failed"})
		return
	}
	enc, err := cred.Encrypt(priv)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encrypt failed"})
		return
	}

	// 若指定 host_uuid，直接创建一条 generated_key 绑定
	bindingID := uint(0)
	if hostUUID != "" {
		var host models.Host
		if err := db.DB.Where("uuid = ? AND tenant_id = ?", hostUUID, tenantID).First(&host).Error; err == nil {
			b := models.ChannelBinding{
				TenantID: tenantID, HostID: host.ID, Type: "ssh",
				Address: host.IP + ":22", AuthMode: "generated_key",
				Username: "root", Credential: enc, Priority: 100, Enabled: false,
			}
			if db.DB.Create(&b).Error == nil {
				bindingID = b.ID
			}
		}
	}

	installCmd := fmt.Sprintf("mkdir -p ~/.ssh && echo '%s' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys", pub)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"public_key":            pub,
		"install_command":       installCmd,
		"channel_binding_id":    bindingID,
		"private_key_encrypted": enc,
	})
}

// runProbe executes a single probe for a binding and persists status.
func runProbe(ctx context.Context, b *models.ChannelBinding, cred services.CredentialService) map[string]interface{} {
	ch, ok := channelReg.Get(b.Type)
	if !ok {
		return map[string]interface{}{"ok": false, "error": "unknown channel type"}
	}
	res, err := ch.Probe(ctx, b, decryptor{cred})
	now := time.Now()
	status := collectors.StatusOK
	if !res.OK {
		if res.Err != "" {
			status = res.Err
		} else if err != nil {
			status = collectors.StatusUnreachable
		}
	}
	db.DB.Model(b).Updates(map[string]interface{}{
		"last_probe_at": now,
		"last_status":   status,
	})
	if res.OK {
		return map[string]interface{}{"ok": true, "os": res.OS, "host": res.Host}
	}
	return map[string]interface{}{"ok": false, "error": status, "detail": errString(err)}
}

func probeChannel(b models.ChannelBinding, cred services.CredentialService) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	runProbe(ctx, &b, cred)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// StartCollectorScheduler launches the background agentless collection loop.
func StartCollectorScheduler() {
	initCollectors()
	if credService == nil {
		log.Println("[scheduler] credential service unavailable, agentless collection disabled")
		return
	}
	go runCollectorScheduler()
}

const (
	schedulerInterval    = 60 * time.Second
	agentActiveWindow    = 5 * time.Minute
	schedulerCollectTO   = 20 * time.Second
	schedulerWorkers      = 8
	channelFailThreshold = 5
)

func runCollectorScheduler() {
	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()
	for range ticker.C {
		collectCycle()
	}
}

func collectCycle() {
	type job struct {
		host    models.Host
		binding models.ChannelBinding
	}
	var jobs []job
	cutoff := time.Now().Add(-agentActiveWindow)
	var hosts []models.Host
	db.DB.Where("last_heartbeat IS NULL OR last_heartbeat < ?", cutoff).Find(&hosts)
	for _, h := range hosts {
		var bs []models.ChannelBinding
		db.DB.Where("host_id = ? AND enabled = ?", h.ID, true).Order("priority asc").Find(&bs)
		if len(bs) == 0 {
			continue
		}
		for _, b := range bs {
			jobs = append(jobs, job{host: h, binding: b})
		}
	}

	if len(jobs) == 0 {
		return
	}
	sem := make(chan struct{}, schedulerWorkers)
	var wg sync.WaitGroup
	byHost := map[uint]chan struct{}{}
	for _, j := range jobs {
		if _, ok := byHost[j.host.ID]; !ok {
			byHost[j.host.ID] = make(chan struct{}, 1)
		}
	}
	for _, j := range jobs {
		j := j
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			collectHost(j.host, j.binding, byHost[j.host.ID])
		}()
	}
	wg.Wait()
}

func collectHost(host models.Host, binding models.ChannelBinding, gate chan struct{}) {
	select {
	case gate <- struct{}{}:
		defer func() { <-gate }()
	default:
		return // 该主机本周期已有采集在跑
	}

	ctx, cancel := context.WithTimeout(context.Background(), schedulerCollectTO)
	defer cancel()

	ch, ok := channelReg.Get(binding.Type)
	if !ok {
		return
	}
	cred := *credService
	res, err := ch.Collect(ctx, &binding, decryptor{cred})
	if err != nil {
		status := classifyErr(err)
		db.DB.Model(&binding).Updates(map[string]interface{}{
			"last_status": status,
			"fail_count":  binding.FailCount + 1,
		})
		binding.FailCount++
		if binding.FailCount >= channelFailThreshold {
			db.DB.Model(&binding).Update("enabled", false)
			db.DB.Create(&models.CollectEvent{
				TenantID: host.TenantID, HostID: host.ID, ChannelID: binding.ID,
				Type: "channel_disabled", Message: fmt.Sprintf("channel %s disabled after %d failures", binding.Type, binding.FailCount),
			})
		}
		return
	}

	now := time.Now()
	sample := models.MetricSample{
		TenantID: host.TenantID, HostID: host.ID, Timestamp: now,
		CPUPercent: res.Metrics.CPUPercent, MemPercent: res.Metrics.MemPercent,
		MemUsed: res.Metrics.MemUsed, MemTotal: res.Metrics.MemTotal,
		DiskUsed: res.Metrics.DiskUsed, DiskTotal: res.Metrics.DiskTotal,
		NetRxBps: res.Metrics.NetRxBps, NetTxBps: res.Metrics.NetTxBps,
		Load1: res.Metrics.Load1, Load5: res.Metrics.Load5, Load15: res.Metrics.Load15,
		UptimeSec: res.Metrics.UptimeSec,
	}
	if err := db.DB.Create(&sample).Error; err != nil {
		log.Printf("[scheduler] save metric failed host=%d: %v", host.ID, err)
		return
	}
	db.DB.Model(&host).Updates(map[string]interface{}{
		"status":         "online",
		"last_heartbeat": now,
	})
	db.DB.Model(&binding).Updates(map[string]interface{}{
		"last_status": collectors.StatusOK,
		"fail_count":  0,
	})
	evaluateMetricAlerts(host, sample)
}

func classifyErr(err error) string {
	msg := err.Error()
	for _, s := range []string{collectors.StatusUnreachable, collectors.StatusAuthFailed, collectors.StatusDenied, collectors.StatusUnsupported, collectors.StatusParseError} {
		if strings.HasPrefix(msg, s) {
			return s
		}
	}
	return collectors.StatusUnreachable
}
