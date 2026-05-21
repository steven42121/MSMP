package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"MSMP/server/config"
	"MSMP/server/db"
	"MSMP/server/models"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ContextUserID   contextKey = "userID"
	ContextTenantID contextKey = "tenantID"
	ContextUserRole contextKey = "userRole"
)

// CORSMiddleware 处理跨域请求
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-Id")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware JWT 认证中间件，跳过公开接口
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 公开接口，无需认证
		publicPaths := []string{
			"/api/health",
			"/api/auth/login",
			"/api/auth/refresh",
			"/api/agents/register",
			"/api/agents/heartbeat",
			"/api/agents/assets",
			"/api/agents/metrics",
		}
		for _, pp := range publicPaths {
			if path == pp {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Agent 认证（使用 AgentToken）
		if strings.HasPrefix(path, "/api/agents/") {
			agentToken := r.Header.Get("Authorization")
			if agentToken != "" {
				agentToken = strings.TrimPrefix(agentToken, "Bearer ")
				var token models.AgentToken
				if err := db.DB.Where("token = ? AND revoked = ?", agentToken, false).First(&token).Error; err == nil {
					if token.ExpiresAt == nil || token.ExpiresAt.After(time.Now()) {
						ctx := context.WithValue(r.Context(), ContextTenantID, token.TenantID)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}
			http.Error(w, `{"error":"invalid agent token"}`, http.StatusUnauthorized)
			return
		}

		// JWT 认证（Web 用户）
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(config.C.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			log.Printf("JWT parse error: %v", err)
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		userID := uint((*claims)["user_id"].(float64))
		tenantID := uint((*claims)["tenant_id"].(float64))
		role := (*claims)["role"].(string)

		ctx := context.WithValue(r.Context(), ContextUserID, userID)
		ctx = context.WithValue(ctx, ContextTenantID, tenantID)
		ctx = context.WithValue(ctx, ContextUserRole, role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getTenantID 从 context 中获取租户 ID
func getTenantID(r *http.Request) uint {
	if v, ok := r.Context().Value(ContextTenantID).(uint); ok {
		return v
	}
	return 0
}

// writeJSON 统一 JSON 响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}