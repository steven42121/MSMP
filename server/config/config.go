package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig
	DB           DBConfig
	JWT          JWTConfig
	Agent        AgentConfig
	Notification NotificationConfig
	Security     SecurityConfig
	Retention    RetentionConfig
}

type SecurityConfig struct {
	CredentialKey      string   `mapstructure:"credentialkey"`
	IPAllowList        []string `mapstructure:"ip_allowlist"`
	AllowedOrigins     []string `mapstructure:"allowed_origins"`
	MaxLoginAttempts   int      `mapstructure:"max_login_attempts"`
	LoginLockoutSec    int      `mapstructure:"login_lockout_sec"`
	RateLimitPerMin    int      `mapstructure:"rate_limit_per_min"`
	PVEInsecureSkipVerify bool     `mapstructure:"pve_insecure_verify"`
}

type RetentionConfig struct {
	RawRetentionDays   int `mapstructure:"raw_retention_days"`   // 原始数据保留天数
	DownsampleAtDays   int `mapstructure:"downsample_at_days"`   // 在此天数之后开始降采样
	DownsampleInterval int `mapstructure:"downsample_interval"`  // 降采样粒度（分钟），默认 5
}

type NotificationConfig struct {
	WebhookURL string
}

type ServerConfig struct {
	Addr              string   `mapstructure:"addr"`
	AdvertiseAddr     string   `mapstructure:"advertise_addr"`
	Mode              string   `mapstructure:"mode"`
	Nodes             []string `mapstructure:"nodes"`
	NodeID            string   `mapstructure:"node_id"`
	MaxRequestBodyBytes int64  `mapstructure:"max_request_body_bytes"`
}

type DBConfig struct {
	Driver          string
	DSN             string
	SqlitePath      string
	MaxOpenConns    int `mapstructure:"max_open_conns"`
	MaxIdleConns    int `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int `mapstructure:"conn_max_lifetime"` // 秒
}

type JWTConfig struct {
	Secret     string
	ExpireHour int
}

type AgentConfig struct {
	Transport      string
	HeartbeatSec   int
	AssetReportSec int
	MetricReportSec int
	OfflineAfterSec int
	LatestVersion  string `mapstructure:"latest_version"`
	DownloadURL    string `mapstructure:"download_url"`
}

var C *Config

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("../config")

	v.SetDefault("server.addr", ":8080")
	v.SetDefault("server.advertise_addr", "")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.nodes", []string{})
	v.SetDefault("server.node_id", "")
	v.SetDefault("server.max_request_body_bytes", 10<<20) // 10MB
	// 默认统一使用 PostgreSQL（多节点共享库）；sqlite 仅用于本地快速验证。
	v.SetDefault("db.driver", "postgres")
	v.SetDefault("db.sqlitepath", "msmp.db")
	v.SetDefault("db.dsn", "host=127.0.0.1 user=msmp password=msmp123 dbname=msmp port=5432 sslmode=disable TimeZone=Asia/Shanghai")
	v.SetDefault("db.max_open_conns", 25)
	v.SetDefault("db.max_idle_conns", 10)
	v.SetDefault("db.conn_max_lifetime", 900) // 15 分钟
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expirehour", 24)
	v.SetDefault("agent.transport", "http")
	v.SetDefault("agent.heartbeatsec", 30)
	v.SetDefault("agent.assetreportsec", 300)
	v.SetDefault("agent.metricreportsec", 60)
	v.SetDefault("agent.offlineaftersec", 120)
	v.SetDefault("agent.latest_version", "0.1.0")
	v.SetDefault("agent.download_url", "https://github.com/steven42121/MSMP/releases/download/{{.Tag}}/msmp-{{.Tag}}-linux-amd64.tar.gz")
	v.SetDefault("notification.webhookurl", "")
	v.SetDefault("security.credentialkey", "")
	v.SetDefault("security.ip_allowlist", []string{})
	v.SetDefault("security.allowed_origins", []string{})
	v.SetDefault("security.max_login_attempts", 5)
	v.SetDefault("security.login_lockout_sec", 600)
	v.SetDefault("security.rate_limit_per_min", 0)
	v.SetDefault("security.pve_insecure_verify", true)
	v.SetDefault("retention.raw_retention_days", 90)
	v.SetDefault("retention.downsample_at_days", 7)
	v.SetDefault("retention.downsample_interval", 5)

	v.SetEnvPrefix("MSMP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	_ = v.ReadInConfig()

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	C = &c
	return &c, nil
}

// Validate 校验生产环境配置安全性。
func (c *Config) Validate() error {
	if c.Server.Mode != "release" {
		return nil
	}
	if c.JWT.Secret == "change-me-in-production" || c.JWT.Secret == "" {
		return fmt.Errorf("refusing to start in release mode: jwt.secret is set to a known default; override MSMP_JWT_SECRET or config file")
	}
	if len(c.JWT.Secret) < 16 {
		return fmt.Errorf("jwt.secret must be at least 16 characters in release mode")
	}
	if strings.Contains(c.DB.DSN, "password=msmp123") {
		return fmt.Errorf("refusing to start in release mode: db.dsn contains default password 'msmp123'; change the database password before deploying")
	}
	return nil
}
