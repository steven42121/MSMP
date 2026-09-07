package controllers

import (
	"sync/atomic"
	"time"
)

// buildInfo 由 main 包在启动时注入（编译时 ldflags 分配的版本号）。
var (
	buildVersion atomic.Value // string
	startTime    = time.Now()
)

// SetBuildVersion 设置构建版本（供 /api/health 返回）。
func SetBuildVersion(v string) {
	if v != "" {
		buildVersion.Store(v)
	}
}

func getBuildVersion() string {
	if v, ok := buildVersion.Load().(string); ok {
		return v
	}
	return ""
}

func getUptimeSeconds() int64 {
	return int64(time.Since(startTime).Seconds())
}