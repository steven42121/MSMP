package services

import (
	"MSMP/server/db"
	"MSMP/server/models"
)

// GetSetting 读取系统设置值
func GetSetting(key string) string {
	var s models.Setting
	if err := db.DB.Where("key = ?", key).First(&s).Error; err == nil {
		return s.Value
	}
	return ""
}
