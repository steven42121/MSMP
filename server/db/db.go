package db

import (
	"fmt"
	"log/slog"
	"time"

	"MSMP/server/config"
	"MSMP/server/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.Config) error {
	var dialector gorm.Dialector
	switch cfg.DB.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.DB.DSN)
	case "sqlite", "":
		dialector = sqlite.Open(cfg.DB.SqlitePath)
	default:
		return fmt.Errorf("unsupported db driver: %s", cfg.DB.Driver)
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}
	DB = gdb

	// 配置连接池
	if sqlDB, err := gdb.DB(); err == nil {
		if cfg.DB.MaxOpenConns > 0 {
			sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
		} else {
			sqlDB.SetMaxOpenConns(25)
		}
		if cfg.DB.MaxIdleConns > 0 {
			sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
		} else {
			sqlDB.SetMaxIdleConns(10)
		}
		if cfg.DB.ConnMaxLifetime > 0 {
			sqlDB.SetConnMaxLifetime(time.Duration(cfg.DB.ConnMaxLifetime) * time.Second)
		} else {
			sqlDB.SetConnMaxLifetime(15 * time.Minute)
		}
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
		slog.Info("DB pool configured", "max_open", sqlDB.Stats().MaxOpenConnections)
	}

	return gdb.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Host{},
		&models.HostTag{},
		&models.AgentToken{},
		&models.AssetSnapshot{},
		&models.MetricSample{},
		&models.MetricDownsample{},
		&models.HostEvent{},
		&models.Task{},
		&models.AlertRule{},
		&models.AuditLog{},
		&models.Setting{},
		&models.ChannelBinding{},
		&models.CollectEvent{},
		&models.AlertSuppression{},
		&models.AlertSilence{},
		&models.AlertEscalation{},
		&models.AvailProbe{},
		&models.CronJob{},
		&models.CronLog{},
		&models.HostProcess{},
		&models.HostPort{},
		&models.HostPackage{},
		&models.PluginInstance{},
		&models.ClusterNode{},
	)
}