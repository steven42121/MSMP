package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"MSMP/server/config"
	"MSMP/server/db"
	"MSMP/server/models"
)

func SettingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var settings []models.Setting
		db.DB.Find(&settings)
		result := map[string]string{}
		for _, s := range settings {
			result[s.Key] = s.Value
		}
		writeJSON(w, http.StatusOK, result)

	case http.MethodPut:
		var updates map[string]string
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		for k, v := range updates {
			var s models.Setting
			if err := db.DB.Where("key = ?", k).First(&s).Error; err != nil {
				db.DB.Create(&models.Setting{Key: k, Value: v})
			} else {
				db.DB.Model(&s).Update("value", v)
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "saved"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// GetSetting 读取设置值，未设置时返回配置默认值
func GetSetting(key string) string {
	var s models.Setting
	if err := db.DB.Where("key = ?", key).First(&s).Error; err == nil {
		return s.Value
	}
	switch key {
	case "notification.webhookurl":
		return configC().Notification.WebhookURL
	case "agent.offlineaftersec":
		return strconv.Itoa(configC().Agent.OfflineAfterSec)
	}
	return ""
}

func configC() *config.Config {
	return config.C
}
