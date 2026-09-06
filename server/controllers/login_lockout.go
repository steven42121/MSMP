package controllers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// loginAttempt 记录单次登录失败尝试。
type loginAttempt struct {
	count    int
	lastFail time.Time
}

var (
	loginLocks     sync.Mutex
	failedAttempts = make(map[string]*loginAttempt)
	rateLock       sync.Mutex
	rateCounters   = make(map[string]*atomic.Int64)
	rateWindows    = make(map[string]time.Time)
)

const (
	rateWindowSec = 60
)

// getRemoteIP 从请求中提取客户端真实 IP（支持代理）。
func getRemoteIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip := strings.TrimSpace(strings.Split(xff, ",")[0])
		if ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ipInAllowList 检查 IP 是否在白名单中（支持 CIDR）。
func ipInAllowList(ipStr string, allowList []string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, entry := range allowList {
		entry = strings.TrimSpace(entry)
		if entry == ipStr {
			return true
		}
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err == nil && cidr.Contains(ip) {
				return true
			}
		}
	}
	return false
}

// checkLoginLockout 检查 IP 是否被锁定。
func checkLoginLockout(ip string, maxAttempts, lockoutSec int) bool {
	loginLocks.Lock()
	defer loginLocks.Unlock()

	now := time.Now()
	attempt, ok := failedAttempts[ip]
	if !ok {
		return false
	}
	if now.Sub(attempt.lastFail) > time.Duration(lockoutSec)*time.Second {
		delete(failedAttempts, ip)
		return false
	}
	return attempt.count >= maxAttempts
}

// recordLoginFailure 记录一次登录失败。
func recordLoginFailure(ip string, maxAttempts, lockoutSec int) {
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

// CleanExpiredAttempts 清理过期的失败记录。
func CleanExpiredAttempts(lockoutSec int) {
	loginLocks.Lock()
	defer loginLocks.Unlock()

	now := time.Now()
	for ip, attempt := range failedAttempts {
		if now.Sub(attempt.lastFail) > time.Duration(lockoutSec)*time.Second {
			delete(failedAttempts, ip)
		}
	}
}

// CheckRateLimit 检查 IP 是否在速率限制窗口内超限。
func CheckRateLimit(ip string, maxPerMin int) bool {
	if maxPerMin <= 0 {
		return true
	}
	now := time.Now()
	rateLock.Lock()
	defer rateLock.Unlock()

	lastWindow, ok := rateWindows[ip]
	if !ok || now.Sub(lastWindow) > time.Duration(rateWindowSec)*time.Second {
		cnt := &atomic.Int64{}
		cnt.Store(1)
		rateCounters[ip] = cnt
		rateWindows[ip] = now
		return true
	}
	cnt := rateCounters[ip]
	if cnt.Add(1) > int64(maxPerMin) {
		return false
	}
	return true
}

// ResetRateLimit 登录后成功时重置计数。
func ResetRateLimit(ip string) {
	rateLock.Lock()
	defer rateLock.Unlock()
	delete(rateCounters, ip)
	delete(rateWindows, ip)
}

// clearLoginAttempts 登录成功后清除该 IP 的失败记录。
func clearLoginAttempts(ip string) {
	loginLocks.Lock()
	defer loginLocks.Unlock()
	delete(failedAttempts, ip)
}
