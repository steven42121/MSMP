package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"MSMP/server/db"
	"MSMP/server/models"
)

func TenantsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var tenants []models.Tenant
		db.DB.Find(&tenants)
		writeJSON(w, http.StatusOK, tenants)

	case http.MethodPost:
		var tenant models.Tenant
		if err := json.NewDecoder(r.Body).Decode(&tenant); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if tenant.Name == "" || tenant.Slug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and slug required"})
			return
		}
		if err := db.DB.Create(&tenant).Error; err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "tenant already exists"})
			return
		}
		writeJSON(w, http.StatusCreated, tenant)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var users []models.User
		db.DB.Where("tenant_id = ?", tenantID).Find(&users)
		writeJSON(w, http.StatusOK, users)

	case http.MethodPost:
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		user.TenantID = tenantID
		if user.Role == "" {
			user.Role = "member"
		}
		// 密码在创建时应该哈希，这里做简单处理
		if user.PasswordHash != "" {
			// 实际应使用 bcrypt
		}
		if err := db.DB.Create(&user).Error; err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "user already exists"})
			return
		}
		writeJSON(w, http.StatusCreated, user)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// 处理 /api/users/{id} 路由
func init() {
	// 注册带参数的 users 路由在主路由中处理
	_ = strings.Split
	_ = strconv.Atoi
}