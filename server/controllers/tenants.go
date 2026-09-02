package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"MSMP/server/db"
	"MSMP/server/models"

	"golang.org/x/crypto/bcrypt"
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

type UserCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type UserUpdateRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var users []models.User
		query := db.DB.Where("tenant_id = ?", tenantID)
		if keyword := r.URL.Query().Get("keyword"); keyword != "" {
			query = query.Where("username LIKE ?", "%"+keyword+"%")
		}
		query.Order("created_at DESC").Find(&users)
		writeJSON(w, http.StatusOK, users)

	case http.MethodPost:
		var req UserCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if req.Username == "" || req.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
			return
		}

		role := req.Role
		if role == "" {
			role = "member"
		}

		user := models.User{
			TenantID:     tenantID,
			Username:     req.Username,
			PasswordHash: string(hash),
			Email:        req.Email,
			Role:         role,
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

func UserDetailHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user id required"})
		return
	}

	userID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	var user models.User
	if err := db.DB.Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, user)
	case http.MethodPut:
		var req UserUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		updates := map[string]interface{}{}
		if req.Email != "" {
			updates["email"] = req.Email
		}
		if req.Role != "" {
			updates["role"] = req.Role
		}
		if req.Password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
				return
			}
			updates["password_hash"] = string(hash)
		}
		if len(updates) > 0 {
			db.DB.Model(&user).Updates(updates)
		}
		db.DB.Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user)
		writeJSON(w, http.StatusOK, user)
	case http.MethodDelete:
		db.DB.Delete(&user)
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
