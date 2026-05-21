package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"MSMP/server/config"
	"MSMP/server/db"
	"MSMP/server/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TenantID uint   `json:"tenant_id"` // 可选，多租户登录时指定
}

type LoginResponse struct {
	Token    string       `json:"token"`
	User     UserInfo     `json:"user"`
	ExpireAt time.Time    `json:"expire_at"`
}

type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	TenantID uint   `json:"tenant_id"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	var user models.User
	query := db.DB.Where("username = ?", req.Username)
	if req.TenantID > 0 {
		query = query.Where("tenant_id = ?", req.TenantID)
	}
	if err := query.First(&user).Error; err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	expireAt := time.Now().Add(time.Duration(config.C.JWT.ExpireHour) * time.Hour)

	claims := jwt.MapClaims{
		"user_id":   user.ID,
		"tenant_id": user.TenantID,
		"username":  user.Username,
		"role":      user.Role,
		"exp":       expireAt.Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.C.JWT.Secret))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token: tokenStr,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			TenantID: user.TenantID,
		},
		ExpireAt: expireAt,
	})
}

func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	// 需要有效 JWT 才能刷新
	userID := r.Context().Value(ContextUserID).(uint)
	tenantID := getTenantID(r)
	role := r.Context().Value(ContextUserRole).(string)

	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "user not found"})
		return
	}

	expireAt := time.Now().Add(time.Duration(config.C.JWT.ExpireHour) * time.Hour)
	claims := jwt.MapClaims{
		"user_id":   user.ID,
		"tenant_id": tenantID,
		"username":  user.Username,
		"role":      role,
		"exp":       expireAt.Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(config.C.JWT.Secret))

	writeJSON(w, http.StatusOK, LoginResponse{
		Token: tokenStr,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			TenantID: tenantID,
		},
		ExpireAt: expireAt,
	})
}

// HealthHandler 健康检查
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := db.DB.DB()
	dbStatus := "ok"
	if err != nil || sqlDB.Ping() != nil {
		dbStatus = "error"
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "running",
		"db":     dbStatus,
	})
}