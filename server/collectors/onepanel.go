package collectors

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"MSMP/server/models"
)

const onepanelTimeout = 15 * time.Second

// OnePanelChannel 通过 1Panel API Key 采集主机实时指标。
// 认证：1Panel-Token = HMAC-SHA256(API-Key, "1panel:"+timestamp)，1Panel-Timestamp 为秒级时间戳。
// API Key 存入渠道凭据（Credential 加密存储），AuthMode 固定为 "api_key"。
type OnePanelChannel struct {
	HTTP *http.Client
}

func (o *OnePanelChannel) Type() string { return "1panel" }

func (o *OnePanelChannel) client() *http.Client {
	if o.HTTP != nil {
		return o.HTTP
	}
	// 1Panel 默认使用自签名证书，采集场景跳过 TLS 校验（仅读监控指标）。
	return &http.Client{
		Timeout: onepanelTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// onepanelSign 生成 1Panel API 签名（HMAC-SHA256 方式）。
func onepanelSign(apiKey, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte("1panel:" + timestamp))
	return hex.EncodeToString(mac.Sum(nil))
}

func (o *OnePanelChannel) do(ctx context.Context, apiKey, addr, method, path string) ([]byte, int, error) {
	base := strings.TrimRight(addr, "/")
	if !strings.HasPrefix(base, "http") {
		base = "http://" + base
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req, err := http.NewRequestWithContext(ctx, method, base+path, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	req.Header.Set("1Panel-Token", onepanelSign(apiKey, ts))
	req.Header.Set("1Panel-Timestamp", ts)
	resp, err := o.client().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

// onepanelResponse 是 1Panel 统一响应结构。
type onepanelResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// onepanelCurrent 对应 1Panel /dashboard/current/:ioOption/:netOption 返回的实时指标。
type onepanelCurrent struct {
	Uptime            uint64  `json:"uptime"`
	CPUUsedPercent    float64 `json:"cpuUsedPercent"`
	Load1             float64 `json:"load1"`
	Load5             float64 `json:"load5"`
	Load15            float64 `json:"load15"`
	MemoryUsed        uint64  `json:"memoryUsed"`
	MemoryTotal       uint64  `json:"memoryTotal"`
	MemoryUsedPercent float64 `json:"memoryUsedPercent"`
	DiskData          []struct {
		Used  uint64 `json:"used"`
		Total uint64 `json:"total"`
	} `json:"diskData"`
}

func (o *OnePanelChannel) Probe(ctx context.Context, binding *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	apiKey, err := cred.Decrypt(binding.Credential)
	if err != nil {
		return ProbeResult{Err: StatusAuthFailed}, err
	}
	body, code, err := o.do(ctx, apiKey, binding.Address, http.MethodGet, "/api/v2/dashboard/current/node")
	if err != nil {
		pr, _ := classify(err)
		return pr, err
	}
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		return ProbeResult{Err: StatusAuthFailed}, fmt.Errorf("%s:status %d", StatusAuthFailed, code)
	}
	if code != http.StatusOK {
		return ProbeResult{Err: StatusUnreachable}, fmt.Errorf("%s:status %d", StatusUnreachable, code)
	}
	var r onepanelResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return ProbeResult{Err: StatusParseError}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	if r.Code == 401 || r.Code == 403 {
		return ProbeResult{Err: StatusAuthFailed}, fmt.Errorf("%s:code %d", StatusAuthFailed, r.Code)
	}
	if r.Code != 200 {
		return ProbeResult{Err: StatusUnreachable}, fmt.Errorf("%s:code %d (%s)", StatusUnreachable, r.Code, r.Message)
	}
	return ProbeResult{OK: true, OS: "linux"}, nil
}

func (o *OnePanelChannel) Collect(ctx context.Context, binding *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	apiKey, err := cred.Decrypt(binding.Credential)
	if err != nil {
		return CollectResult{}, err
	}
	body, code, err := o.do(ctx, apiKey, binding.Address, http.MethodGet, "/api/v2/dashboard/current/all/all")
	if err != nil {
		return CollectResult{}, err
	}
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		return CollectResult{}, fmt.Errorf("%s:status %d", StatusAuthFailed, code)
	}
	if code != http.StatusOK {
		return CollectResult{}, fmt.Errorf("%s:status %d", StatusUnreachable, code)
	}
	var r onepanelResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	if r.Code != 200 {
		return CollectResult{}, fmt.Errorf("%s:code %d (%s)", StatusUnreachable, r.Code, r.Message)
	}
	var cur onepanelCurrent
	if err := json.Unmarshal(r.Data, &cur); err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}

	var diskUsed, diskTotal uint64
	for _, d := range cur.DiskData {
		diskUsed += d.Used
		diskTotal += d.Total
	}

	return CollectResult{
		Metrics: MetricDataLike{
			CPUPercent: cur.CPUUsedPercent,
			MemPercent: cur.MemoryUsedPercent,
			MemUsed:    cur.MemoryUsed,
			MemTotal:   cur.MemoryTotal,
			DiskUsed:   diskUsed,
			DiskTotal:  diskTotal,
			Load1:      cur.Load1,
			Load5:      cur.Load5,
			Load15:     cur.Load15,
			UptimeSec:  cur.Uptime,
		},
		Missing:  []string{"net_rx_bps", "net_tx_bps"},
		Duration: time.Since(start),
	}, nil
}
