// Package metrics 提供 Prometheus 格式的服务器自监控指标。
package metrics

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Stats 汇总服务器运行指标。
type Stats struct {
	Start        time.Time
	RequestCount atomic.Int64
	ErrorCount   atomic.Int64
	// 主机计数（每秒查询 DB 并写入，由上层填充）
	HostCount    atomic.Int64
	AgentCount   atomic.Int64
	MetricCount  atomic.Int64
}

var Global = &Stats{Start: time.Now()}

// ObserveRequest 每次 HTTP 请求调用一次（由中间件调用）。
func (s *Stats) ObserveRequest(isError bool) {
	s.RequestCount.Add(1)
	if isError {
		s.ErrorCount.Add(1)
	}
}

// FormatPrometheus 将当前统计格式化为 Prometheus exposition 文本。
func (s *Stats) FormatPrometheus(w io.Writer) {
	uptime := time.Since(s.Start).Seconds()

	// ── process-level ──────────────────────────────────────
	fmt.Fprintf(w, "# HELP msmp_process_uptime_seconds 服务器运行时间（秒）\n")
	fmt.Fprintf(w, "# TYPE msmp_process_uptime_seconds gauge\n")
	fmt.Fprintf(w, "msmp_process_uptime_seconds %.2f\n", uptime)

	fmt.Fprintf(w, "# HELP msmp_process_goroutines 当前 Goroutine 数\n")
	fmt.Fprintf(w, "# TYPE msmp_process_goroutines gauge\n")
	fmt.Fprintf(w, "msmp_process_goroutines %d\n", runtime.NumGoroutine())

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Fprintf(w, "# HELP msmp_process_mem_alloc_bytes 堆内存分配量\n")
	fmt.Fprintf(w, "# TYPE msmp_process_mem_alloc_bytes gauge\n")
	fmt.Fprintf(w, "msmp_process_mem_alloc_bytes %d\n", mem.Alloc)

	// ── request counters ───────────────────────────────────
	fmt.Fprintf(w, "# HELP msmp_requests_total 总请求数\n")
	fmt.Fprintf(w, "# TYPE msmp_requests_total counter\n")
	fmt.Fprintf(w, "msmp_requests_total %d\n", s.RequestCount.Load())

	fmt.Fprintf(w, "# HELP msmp_errors_total 总错误请求数\n")
	fmt.Fprintf(w, "# TYPE msmp_errors_total counter\n")
	fmt.Fprintf(w, "msmp_errors_total %d\n", s.ErrorCount.Load())

	// ── resource gauges ────────────────────────────────────
	fmt.Fprintf(w, "# HELP msmp_hosts_total 主机总数\n")
	fmt.Fprintf(w, "# TYPE msmp_hosts_total gauge\n")
	fmt.Fprintf(w, "msmp_hosts_total %d\n", s.HostCount.Load())

	fmt.Fprintf(w, "# HELP msmp_agents_total 在线 Agent 数\n")
	fmt.Fprintf(w, "# TYPE msmp_agents_total gauge\n")
	fmt.Fprintf(w, "msmp_agents_total %d\n", s.AgentCount.Load())

	fmt.Fprintf(w, "# HELP msmp_metric_samples_total 已存储的指标采样总数\n")
	fmt.Fprintf(w, "# TYPE msmp_metric_samples_total gauge\n")
	fmt.Fprintf(w, "msmp_metric_samples_total %d\n", s.MetricCount.Load())

	// ── cluster ───────────────────────────────────────────
	fmt.Fprintf(w, "# HELP msmp_cluster_node_id 节点身份（label 固定为本地节点）\n")
	fmt.Fprintf(w, "# TYPE msmp_cluster_node_id gauge\n")
	fmt.Fprintf(w, "msmp_cluster_node_id{node=\"local\"} 1\n")

	// ── help footer ────────────────────────────────────────
	fmt.Fprintf(w, "# EOF\n")
}

// formatCounter formats a counter with start + rate.
func formatCounter(w io.Writer, name, help string, current int64, since time.Time) {
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s counter\n", name)
	if since.IsZero() {
		since = time.Now()
	}
	rate := float64(current) / time.Since(since).Seconds()
	fmt.Fprintf(w, "%s %.2f\n", name, rate)
}

// Registry 用于收集可观测计数器，支持并发安全。
type Registry struct {
	mu     sync.RWMutex
	counters map[string]*atomic.Int64
}

func NewRegistry() *Registry {
	return &Registry{counters: make(map[string]*atomic.Int64)}
}

func (r *Registry) Inc(key string) {
	r.mu.Lock()
	c, ok := r.counters[key]
	if !ok {
		c = &atomic.Int64{}
		r.counters[key] = c
	}
	r.mu.Unlock()
	c.Add(1)
}

func (r *Registry) WritePrometheus(w io.Writer) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for key, c := range r.counters {
		fmt.Fprintf(w, "# HELP msmp_custom_%s %s\n", key, key)
		fmt.Fprintf(w, "# TYPE msmp_custom_%s counter\n", key)
		fmt.Fprintf(w, "msmp_custom_%s %d\n", key, c.Load())
	}
}
