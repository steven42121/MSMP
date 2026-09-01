package db

import (
	"fmt"

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

	return gdb.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Host{},
		&models.HostTag{},
		&models.AgentToken{},
		&models.AssetSnapshot{},
		&models.MetricSample{},
		&models.HostEvent{},
		&models.Task{},
		&models.AlertRule{},
		&models.AuditLog{},
		&models.Setting{},
		&models.ChannelBinding{},
		&models.CollectEvent{},
	)
}