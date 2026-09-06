package config

import (
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
}

type SecurityConfig struct {
	CredentialKey string
}

type NotificationConfig struct {
	WebhookURL string
}

type ServerConfig struct {
	Addr     string   `mapstructure:"addr"`
	Mode     string   `mapstructure:"mode"`
	Nodes    []string `mapstructure:"nodes"`
	NodeID   string   `mapstructure:"node_id"`
}

type DBConfig struct {
	Driver     string
	DSN        string
	SqlitePath string
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
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.nodes", []string{})
	v.SetDefault("server.node_id", "")
	v.SetDefault("db.driver", "sqlite")
	v.SetDefault("db.sqlitepath", "msmp.db")
	v.SetDefault("db.dsn", "host=localhost user=msmp password=msmp dbname=msmp port=5432 sslmode=disable TimeZone=Asia/Shanghai")
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

	v.SetEnvPrefix("MSMP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	_ = v.ReadInConfig()

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	C = &c
	return &c, nil
}
