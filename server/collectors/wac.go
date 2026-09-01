package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"MSMP/server/models"
)

const wacTimeout = 15 * time.Second

type WACChannel struct {
	HTTP *http.Client
}

func (w *WACChannel) Type() string { return "wac" }

func (w *WACChannel) client() *http.Client {
	if w.HTTP != nil {
		return w.HTTP
	}
	return &http.Client{Timeout: wacTimeout}
}

func (w *WACChannel) do(ctx context.Context, b *models.ChannelBinding, secret, path string) ([]byte, int, error) {
	base := strings.TrimRight(b.Address, "/")
	url := base + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	switch b.AuthMode {
	case "gateway":
		req.Header.Set("Authorization", "Bearer "+secret)
	default:
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := w.client().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

func (w *WACChannel) Probe(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	secret, err := cred.Decrypt(b.Credential)
	if err != nil {
		return ProbeResult{Err: StatusAuthFailed}, err
	}
	body, code, err := w.do(ctx, b, secret, "/api/hosts")
	_ = body
	if err != nil {
		pr, _ := classify(err)
		return pr, err
	}
	switch code {
	case 200:
		return ProbeResult{OK: true, OS: "windows"}, nil
	case 401:
		return ProbeResult{Err: StatusAuthFailed}, fmt.Errorf("%s:unauthorized", StatusAuthFailed)
	case 403:
		return ProbeResult{Err: StatusDenied}, fmt.Errorf("%s:forbidden", StatusDenied)
	default:
		return ProbeResult{Err: StatusUnreachable}, fmt.Errorf("%s:status %d", StatusUnreachable, code)
	}
}

func (w *WACChannel) Collect(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	secret, err := cred.Decrypt(b.Credential)
	if err != nil {
		return CollectResult{}, err
	}
	body, code, err := w.do(ctx, b, secret, "/api/metrics")
	if err != nil {
		return CollectResult{}, err
	}
	if code != 200 {
		switch code {
		case 401:
			return CollectResult{}, fmt.Errorf("%s:unauthorized", StatusAuthFailed)
		case 403:
			return CollectResult{}, fmt.Errorf("%s:forbidden", StatusDenied)
		default:
			return CollectResult{}, fmt.Errorf("%s:status %d", StatusUnreachable, code)
		}
	}

	var raw struct {
		CPUPercent  float64 `json:"cpu_percent"`
		MemTotal    uint64  `json:"mem_total"`
		MemUsed     uint64  `json:"mem_used"`
		DiskUsed    uint64  `json:"disk_used"`
		DiskTotal   uint64  `json:"disk_total"`
		NetRxBps    uint64  `json:"net_rx_bps"`
		NetTxBps    uint64  `json:"net_tx_bps"`
		Load1       float64 `json:"load1"`
		Load5       float64 `json:"load5"`
		Load15      float64 `json:"load15"`
		UptimeSec   uint64  `json:"uptime_sec"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	memPct := 0.0
	if raw.MemTotal > 0 {
		memPct = float64(raw.MemUsed) / float64(raw.MemTotal) * 100
	}
	missing := []string{}
	if raw.NetRxBps == 0 && raw.NetTxBps == 0 {
		missing = append(missing, "net_rx_bps", "net_tx_bps")
	}
	return CollectResult{
		Metrics: MetricDataLike{
			CPUPercent: raw.CPUPercent,
			MemPercent: memPct,
			MemUsed:    raw.MemUsed,
			MemTotal:   raw.MemTotal,
			DiskUsed:   raw.DiskUsed,
			DiskTotal:  raw.DiskTotal,
			NetRxBps:   raw.NetRxBps,
			NetTxBps:   raw.NetTxBps,
			Load1:      raw.Load1,
			Load5:      raw.Load5,
			Load15:     raw.Load15,
			UptimeSec:  raw.UptimeSec,
		},
		Missing:  missing,
		Duration: time.Since(start),
	}, nil
}
