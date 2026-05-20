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
	ID        uint      `gorm:"primaryKey" json:"id"`
	TenantID  uint      `gorm:"index;not null" json:"tenant_id"`
	HostID    uint      `gorm:"index" json:"host_id"`
	Type      string    `gorm:"size:32;index" json:"type"`
	Level     string    `gorm:"size:16" json:"level"`
	Message   string    `gorm:"type:text" json:"message"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}
