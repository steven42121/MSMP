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
	ContextUsername contextKey = "username"
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

		// IP 白名单检查（配置非空时生效，支持 CIDR）
		if len(config.C.Security.IPAllowList) > 0 {
			clientIP := getRemoteIP(r)
			if !ipInAllowList(clientIP, config.C.Security.IPAllowList) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "IP 不在白名单中"})
				return
			}
		}

		// 公开接口，无需认证
		publicPaths := []string{
			"/api/health",
			"/api/auth/login",
			"/api/auth/refresh",
			"/api/agents/register", // 注册使用一次性 token，需单独验证
			"/api/cluster/info",
			"/api/cluster/ping",
			"/api/cluster/leader",
			"/metrics",
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
		// WebSocket 无法携带 header，允许通过 query 参数传递 token
		if authHeader == "" {
			if q := r.URL.Query().Get("token"); q != "" {
				authHeader = "Bearer " + q
			}
		}
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

		userIDFloat, _ := (*claims)["user_id"].(float64)
		tenantIDFloat, _ := (*claims)["tenant_id"].(float64)
		userRole, _ := (*claims)["role"].(string)
		username, _ := (*claims)["username"].(string)

		userID := uint(userIDFloat)
		tenantID := uint(tenantIDFloat)

		ctx := context.WithValue(r.Context(), ContextUserID, userID)
		ctx = context.WithValue(ctx, ContextTenantID, tenantID)
		ctx = context.WithValue(ctx, ContextUserRole, userRole)
		ctx = context.WithValue(ctx, ContextUsername, username)

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

// getUserID 从 context 中获取用户 ID
func getUserID(r *http.Request) uint {
	if v, ok := r.Context().Value(ContextUserID).(uint); ok {
		return v
	}
	return 0
}

// getUsername 从 context 中获取用户名
func getUsername(r *http.Request) string {
	if v, ok := r.Context().Value(ContextUsername).(string); ok {
		return v
	}
	return ""
}

// getRole 从 context 中获取角色
func getRole(r *http.Request) string {
	if v, ok := r.Context().Value(ContextUserRole).(string); ok {
		return v
	}
	return ""
}

// RequireRole 仅允许指定角色访问，否则返回 403
func RequireRole(roles []string, next http.HandlerFunc) http.HandlerFunc {
	allowed := map[string]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowed[getRole(r)] {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden: insufficient role"})
			return
		}
		next(w, r)
	}
}

// statusRecorder 记录响应状态码
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Audit 包装 handler，记录写操作审计日志
func Audit(action, resource string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 仅记录写操作
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete:
		default:
			next(w, r)
			return
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)

		if action != "" {
			tenantID := getTenantID(r)
			log := models.AuditLog{
				TenantID: tenantID,
				UserID:   getUserID(r),
				Username: getUsername(r),
				Action:   action,
				Resource: resource + " " + r.URL.Path,
				Method:   r.Method,
				Status:   rec.status,
				IP:       r.RemoteAddr,
			}
			db.DB.Create(&log)
		}
	}
}

// writeJSON 统一 JSON 响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}
