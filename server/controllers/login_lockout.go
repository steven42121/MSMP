package controllers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// loginAttempt 记录单次登录失败尝试。
type loginAttempt struct {
	count    int
	lastFail time.Time
}

var (
	loginLocks sync.Mutex
	// failedAttempts[ip] 记录该 IP 的登录失败次数和时间
	failedAttempts = make(map[string]*loginAttempt)
)

const (
	// defaultMaxAttempts 允许的最大失败次数
	defaultMaxAttempts = 5
	// defaultLockoutSeconds 锁定时长
	defaultLockoutSeconds = 10 * 60
)

// getRemoteIP 从请求中提取客户端真实 IP（支持代理）。
func getRemoteIP(r *http.Request) string {
	// 优先从 X-Forwarded-For 读取
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	// X-Real-IP
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// 直连
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// checkLoginLockout 检查 IP 是否被锁定。
// 返回 locked=true 时调用方应拒绝请求。
func checkLoginLockout(ip string, maxAttempts, lockoutSeconds int) bool {
	loginLocks.Lock()
	defer loginLocks.Unlock()

	now := time.Now()
	attempt, ok := failedAttempts[ip]
	if !ok {
		return false
	}

	// 过期则清除
	if now.Sub(attempt.lastFail) > time.Duration(lockoutSeconds)*time.Second {
		delete(failedAttempts, ip)
		return false
	}
	return attempt.count >= maxAttempts
}

// recordLoginFailure 记录一次登录失败。
func recordLoginFailure(ip string, maxAttempts, lockoutSeconds int) {
	loginLocks.Lock()
	defer loginLocks.Unlock()

	attempt, ok := failedAttempts[ip]
	if !ok {
		attempt = &loginAttempt{}
		failedAttempts[ip] = attempt
	}
	attempt.count++
	attempt.lastFail = time.Now()
}

// CleanExpiredAttempts 清理过期的失败记录（定期调用）。
func CleanExpiredAttempts(lockoutSeconds int) {
	loginLocks.Lock()
	defer loginLocks.Unlock()

	now := time.Now()
	for ip, attempt := range failedAttempts {
		if now.Sub(attempt.lastFail) > time.Duration(lockoutSeconds)*time.Second {
			delete(failedAttempts, ip)
		}
	}
}
