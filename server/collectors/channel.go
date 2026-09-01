package collectors

import (
	"context"
	"time"

	"MSMP/server/models"
)

const (
	StatusOK           = "ok"
	StatusUnreachable  = "unreachable"
	StatusAuthFailed   = "auth_failed"
	StatusDenied       = "denied"
	StatusUnsupported  = "unsupported"
	StatusParseError   = "parse_error"
)

type MetricDataLike struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	MemUsed    uint64  `json:"mem_used"`
	MemTotal   uint64  `json:"mem_total"`
	DiskUsed   uint64  `json:"disk_used"`
	DiskTotal  uint64  `json:"disk_total"`
	NetRxBps   uint64  `json:"net_rx_bps"`
	NetTxBps   uint64  `json:"net_tx_bps"`
	Load1      float64 `json:"load1"`
	Load5      float64 `json:"load5"`
	Load15     float64 `json:"load15"`
	UptimeSec  uint64  `json:"uptime_sec"`
}

type ProbeResult struct {
	OK   bool
	OS   string
	Host string
	Err  string
}

type CollectResult struct {
	Metrics  MetricDataLike
	Missing  []string
	Duration time.Duration
}

type CredentialProvider interface {
	Decrypt(ciphertext string) (string, error)
}

type Channel interface {
	Type() string
	Probe(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error)
	Collect(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (CollectResult, error)
}

type Registry struct {
	channels map[string]Channel
}

func NewRegistry() *Registry {
	return &Registry{channels: make(map[string]Channel)}
}

func (r *Registry) Register(ch Channel) {
	r.channels[ch.Type()] = ch
}

func (r *Registry) Get(typ string) (Channel, bool) {
	ch, ok := r.channels[typ]
	return ch, ok
}
