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

const baotaTimeout = 15 * time.Second

type BaoTaChannel struct {
	HTTP *http.Client
}

func (b *BaoTaChannel) Type() string { return "baota" }

func (b *BaoTaChannel) client() *http.Client {
	if b.HTTP != nil {
		return b.HTTP
	}
	return &http.Client{Timeout: baotaTimeout}
}

func (b *BaoTaChannel) do(ctx context.Context, binding *models.ChannelBinding, secret, path string) ([]byte, int, error) {
	base := strings.TrimRight(binding.Address, "/")
	url := base + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	req.Header.Set("Authorization", binding.Username+":"+secret)
	resp, err := b.client().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

func (b *BaoTaChannel) Probe(ctx context.Context, binding *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	secret, err := cred.Decrypt(binding.Credential)
	if err != nil {
		return ProbeResult{Err: StatusAuthFailed}, err
	}
	body, code, err := b.do(ctx, binding, secret, "/system?action=GetSystemTotal")
	if err != nil {
		pr, _ := classify(err)
		return pr, err
	}
	if code != 200 {
		switch code {
		case 401:
			return ProbeResult{Err: StatusAuthFailed}, fmt.Errorf("%s:unauthorized", StatusAuthFailed)
		case 403:
			return ProbeResult{Err: StatusDenied}, fmt.Errorf("%s:forbidden", StatusDenied)
		default:
			return ProbeResult{Err: StatusUnreachable}, fmt.Errorf("%s:status %d", StatusUnreachable, code)
		}
	}
	var r struct {
		Status bool `json:"status"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return ProbeResult{Err: StatusParseError}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	if !r.Status {
		return ProbeResult{Err: StatusAuthFailed}, fmt.Errorf("%s:panel rejected", StatusAuthFailed)
	}
	return ProbeResult{OK: true, OS: "linux"}, nil
}

func (b *BaoTaChannel) Collect(ctx context.Context, binding *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	secret, err := cred.Decrypt(binding.Credential)
	if err != nil {
		return CollectResult{}, err
	}
	body, code, err := b.do(ctx, binding, secret, "/system?action=GetNetWork")
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
		Status bool `json:"status"`
		Data   struct {
			CPU     float64 `json:"cpu"`
			Mem     struct {
				MemTotal uint64 `json:"memTotal"`
				MemReal  uint64 `json:"memRealUsage"`
			} `json:"mem"`
			Load    struct {
				One   float64 `json:"one"`
				Five  float64 `json:"five"`
				Fifteen float64 `json:"fifteen"`
			} `json:"load"`
			UpTime  string `json:"upTime"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	if !raw.Status {
		return CollectResult{}, fmt.Errorf("%s:panel status false", StatusParseError)
	}

	memPct := 0.0
	if raw.Data.Mem.MemTotal > 0 {
		memPct = float64(raw.Data.Mem.MemReal) / float64(raw.Data.Mem.MemTotal) * 100
	}

	return CollectResult{
		Metrics: MetricDataLike{
			CPUPercent: raw.Data.CPU,
			MemPercent: memPct,
			MemUsed:    raw.Data.Mem.MemReal,
			MemTotal:   raw.Data.Mem.MemTotal,
			Load1:      raw.Data.Load.One,
			Load5:      raw.Data.Load.Five,
			Load15:     raw.Data.Load.Fifteen,
		},
		Missing:  []string{"disk_used", "disk_total", "net_rx_bps", "net_tx_bps", "uptime_sec"},
		Duration: time.Since(start),
	}, nil
}
