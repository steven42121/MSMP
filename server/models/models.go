package models

import (
	"time"

	"gorm.io/gorm"
)

type Tenant struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Slug      string         `gorm:"size:64;uniqueIndex;not null" json:"slug"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	TenantID     uint           `gorm:"index;not null" json:"tenant_id"`
	Username     string         `gorm:"size:64;not null;uniqueIndex:idx_tenant_user,priority:2" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Email        string         `gorm:"size:128" json:"email"`
	Role         string         `gorm:"size:32;not null;default:'member'" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Host struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	TenantID      uint           `gorm:"index;not null" json:"tenant_id"`
	UUID          string         `gorm:"size:64;uniqueIndex;not null" json:"uuid"`
	Hostname      string         `gorm:"size:128;index" json:"hostname"`
	OS            string         `gorm:"size:32;index" json:"os"`
	OSVersion     string         `gorm:"size:64" json:"os_version"`
	Arch          string         `gorm:"size:16" json:"arch"`
	IP            string         `gorm:"size:64;index" json:"ip"`
	PublicIP      string         `gorm:"size:64" json:"public_ip"`
	CPUModel      string         `gorm:"size:128" json:"cpu_model"`
	CPUCores      int            `json:"cpu_cores"`
	MemoryTotal   uint64         `json:"memory_total"`
	DiskTotal     uint64         `json:"disk_total"`
	AgentVersion  string         `gorm:"size:32" json:"agent_version"`
	Status        string         `gorm:"size:16;index;default:'pending'" json:"status"`
	LastHeartbeat *time.Time     `json:"last_heartbeat"`
	RegisteredAt  time.Time      `json:"registered_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type HostTag struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	HostID   uint   `gorm:"index;not null" json:"host_id"`
	TenantID uint   `gorm:"index;not null" json:"tenant_id"`
	Key      string `gorm:"size:64;not null" json:"key"`
	Value    string `gorm:"size:128" json:"value"`
}

type AgentToken struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	TenantID    uint       `gorm:"index;not null" json:"tenant_id"`
	HostID      *uint      `gorm:"index" json:"host_id,omitempty"`
	Token       string     `gorm:"size:128;uniqueIndex;not null" json:"token"`
	Description string     `gorm:"size:255" json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Revoked     bool       `gorm:"default:false" json:"revoked"`
	CreatedAt   time.Time  `json:"created_at"`
}

type AssetSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TenantID  uint      `gorm:"index;not null" json:"tenant_id"`
	HostID    uint      `gorm:"index;not null" json:"host_id"`
	Payload   string    `gorm:"type:text" json:"payload"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

type MetricSample struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TenantID   uint      `gorm:"index;not null" json:"tenant_id"`
	HostID     uint      `gorm:"index:idx_host_ts,priority:1;not null" json:"host_id"`
	Timestamp  time.Time `gorm:"index:idx_host_ts,priority:2;not null" json:"timestamp"`
	CPUPercent float64   `json:"cpu_percent"`
	MemPercent float64   `json:"mem_percent"`
	MemUsed    uint64    `json:"mem_used"`
	MemTotal   uint64    `json:"mem_total"`
	DiskUsed   uint64    `json:"disk_used"`
	DiskTotal  uint64    `json:"disk_total"`
	NetRxBps   uint64    `json:"net_rx_bps"`
	NetTxBps   uint64    `json:"net_tx_bps"`
	Load1      float64   `json:"load1"`
	Load5      float64   `json:"load5"`
	Load15     float64   `json:"load15"`
	UptimeSec  uint64    `json:"uptime_sec"`
}

type HostEvent struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	TenantID     uint       `gorm:"index;not null" json:"tenant_id"`
	HostID       uint       `gorm:"index" json:"host_id"`
	Type         string     `gorm:"size:32;index" json:"type"`
	Level        string     `gorm:"size:16" json:"level"`
	Message      string     `gorm:"type:text" json:"message"`
	Acknowledged bool       `gorm:"default:false" json:"acknowledged"`
	SilencedUntil *time.Time `json:"silenced_until"`
	CreatedAt    time.Time  `gorm:"index" json:"created_at"`
}

type AlertRule struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	TenantID    uint           `gorm:"index;not null" json:"tenant_id"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Metric      string         `gorm:"size:32;not null" json:"metric"`   // cpu, mem, disk
	Operator    string         `gorm:"size:8;not null" json:"operator"`  // gt, gte, lt, lte
	Threshold   float64        `gorm:"not null" json:"threshold"`
	Level       string         `gorm:"size:16;not null;default:'warning'" json:"level"`
	Enabled     bool           `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TenantID  uint      `gorm:"index" json:"tenant_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Username  string    `gorm:"size:64" json:"username"`
	Action    string    `gorm:"size:64;index" json:"action"`
	Resource  string    `gorm:"size:128" json:"resource"`
	Method    string    `gorm:"size:16" json:"method"`
	Status    int       `json:"status"`
	IP        string    `gorm:"size:64" json:"ip"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

type Setting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:64;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Task struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	TenantID   uint           `gorm:"index;not null" json:"tenant_id"`
	HostID     uint           `gorm:"index;not null" json:"host_id"`
	Type       string         `gorm:"size:32;not null;default:'shell'" json:"type"` // shell, restart, upgrade
	Command    string         `gorm:"type:text" json:"command"`
	Status     string         `gorm:"size:16;index;not null;default:'pending'" json:"status"` // pending, running, success, failed, canceled
	Result     string         `gorm:"type:text" json:"result"`
	TimeoutSec int            `gorm:"default:0" json:"timeout_sec"`
	CreatedBy  uint           `gorm:"index" json:"created_by"`
	StartedAt  *time.Time     `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at"`
	CreatedAt  time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type ChannelBinding struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	TenantID    uint           `gorm:"index;not null" json:"tenant_id"`
	HostID      uint           `gorm:"index;not null" json:"host_id"`
	Type        string         `gorm:"size:16;not null" json:"type"`     // ssh | wac | baota
	Address     string         `gorm:"size:255;not null" json:"address"` // host:port 或网关/面板 URL
	AuthMode    string         `gorm:"size:16;not null" json:"auth_mode"` // password | private_key | generated_key | api_key | gateway
	Username    string         `gorm:"size:64" json:"username"`
	Credential  string         `gorm:"type:text" json:"-"`               // AES-256-GCM 密文 base64，响应中永不返回
	Priority    int            `gorm:"default:100" json:"priority"`       // 数字小优先
	Enabled     bool           `gorm:"default:true" json:"enabled"`
	LastProbeAt *time.Time     `json:"last_probe_at"`
	LastStatus  string         `gorm:"size:32" json:"last_status"` // ok | unreachable | auth_failed | denied | unsupported | parse_error
	FailCount   int            `gorm:"default:0" json:"fail_count"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type CollectEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TenantID  uint      `gorm:"index;not null" json:"tenant_id"`
	HostID    uint      `gorm:"index;not null" json:"host_id"`
	ChannelID uint      `gorm:"index" json:"channel_id"`
	Type      string    `gorm:"size:32;index" json:"type"` // collect_failed | channel_disabled
	Message   string    `gorm:"type:text" json:"message"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// AlertSuppression 告警抑制规则：同类型告警在时间窗口内只发一次
type AlertSuppression struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TenantID      uint      `gorm:"index;not null" json:"tenant_id"`
	Name          string    `gorm:"size:128" json:"name"`
	HostID        uint      `gorm:"index" json:"host_id"`          // 0 = 全局
	Metric        string    `gorm:"size:32" json:"metric"`         // cpu, mem, disk, offline
	Level         string    `gorm:"size:16" json:"level"`          // warning, critical
	WindowMinutes int       `gorm:"default:30" json:"window_minutes"` // 抑制窗口（分钟）
	Enabled       bool      `gorm:"default:true" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AlertSilence 告警静默规则：按时间/主机/标签静默告警
type AlertSilence struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	TenantID     uint       `gorm:"index;not null" json:"tenant_id"`
	Name         string     `gorm:"size:128" json:"name"`
	HostID       uint       `gorm:"index" json:"host_id"`           // 0 = 全局
	LabelKey     string     `gorm:"size:64" json:"label_key"`       // 标签键
	LabelValue   string     `gorm:"size:128" json:"label_value"`    // 标签值
	Level        string     `gorm:"size:16" json:"level"`           // warning, critical, 空=全部
	StartAt      time.Time  `gorm:"index" json:"start_at"`          // 开始时间
	EndAt        time.Time  `gorm:"index" json:"end_at"`            // 结束时间
	CreatorID    uint       `json:"creator_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// AlertEscalation 告警升级规则：未确认告警到达时限后自动升级
type AlertEscalation struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TenantID       uint      `gorm:"index;not null" json:"tenant_id"`
	Name           string    `gorm:"size:128" json:"name"`
	TriggerLevel   string    `gorm:"size:16" json:"trigger_level"`    // warning, critical
	NotifyAfterMin int       `gorm:"default:60" json:"notify_after_min"` // 未确认N分钟后升级
	RetryCount     int       `gorm:"default:3" json:"retry_count"`      // 最大重试次数
	Enabled        bool      `gorm:"default:true" json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
