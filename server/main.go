package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"MSMP/server/clustering"
	"MSMP/server/config"
	"MSMP/server/controllers"
	"MSMP/server/db"
	"MSMP/server/metrics"
	"MSMP/server/models"
	"MSMP/server/services"
)

func main() {
	log.Println("MSMP Server Starting...")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Config loaded: server=%s, db_driver=%s", cfg.Server.Addr, cfg.DB.Driver)

	// 初始化数据库
	if err := db.Init(cfg); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	log.Println("Database initialized successfully")

	// 初始化凭证服务（WebSSH 等场景使用）
	if credSvc, err := services.NewCredentialService(cfg); err != nil {
		log.Printf("Warning: credential service init failed: %v (SSH features disabled)", err)
	} else {
		services.GlobalCredSvc = credSvc
		log.Println("Credential service initialized")
	}

	// 初始化集群状态
	clusterState := clustering.NewClusterState(cfg)

	// 启动离线检测
	go controllers.StartOfflineChecker(cfg.Agent.OfflineAfterSec)

	// 启动告警升级检查器
	go controllers.StartEscalationChecker()

	// 启动时序数据降采样与保留策略清理
	controllers.StartDownsampleCleanup()

	// 启动无 Agent 采集调度器（仅 leader 执行）
	if cfg.Server.Nodes != nil && len(cfg.Server.Nodes) > 0 && !clusterState.IsLeader() {
		log.Println("[cluster] this node is follower, skipping CollectorScheduler")
	} else {
		controllers.StartCollectorScheduler()
	}

	// follower 节点启动心跳循环
	if clusterState.Mode() == "cluster" && !clusterState.IsLeader() {
		go clusterState.StartHeartbeatLoop()
	}

	// 注册路由
	mux := http.NewServeMux()

	// Agent 相关接口
	mux.HandleFunc("/api/agents/register", controllers.AgentRegisterHandler)
	mux.HandleFunc("/api/agents/heartbeat", controllers.AgentHeartbeatHandler)
	mux.HandleFunc("/api/agents/assets", controllers.AgentAssetReportHandler)
	mux.HandleFunc("/api/agents/metrics", controllers.AgentMetricReportHandler)
	mux.HandleFunc("/api/agents/tasks/", controllers.AgentTaskHandler)
	mux.HandleFunc("/api/agents/upgrade/", controllers.AgentUpgradeHandler)

	// 认证接口
	mux.HandleFunc("/api/auth/login", controllers.LoginHandler)
	mux.HandleFunc("/api/auth/refresh", controllers.RefreshTokenHandler)

	// 主机管理接口
	mux.HandleFunc("/api/hosts", controllers.Audit("manage", "host", controllers.HostsHandler))
	mux.HandleFunc("/api/hosts/", controllers.HostDetailHandler)

	// 采集渠道管理接口（无 Agent 采集）
	mux.HandleFunc("/api/channels/ssh-keypair", controllers.RequireRole([]string{"admin"}, controllers.SSHKeypairHandler))
	mux.HandleFunc("/api/channels/", controllers.RequireRole([]string{"admin"}, controllers.ChannelDetailHandler))

	// 监控数据接口
	mux.HandleFunc("/api/metrics", controllers.MetricsHandler)

	// 任务管理接口
	mux.HandleFunc("/api/tasks", controllers.Audit("manage", "task", controllers.TasksHandler))
	mux.HandleFunc("/api/tasks/", controllers.TaskDetailHandler)

	// 事件/告警接口
	mux.HandleFunc("/api/alerts", controllers.AlertsHandler)
	mux.HandleFunc("/api/alerts/", controllers.AlertDetailHandler)
	mux.HandleFunc("/api/alert-rules", controllers.Audit("manage", "alert_rule", controllers.RequireRole([]string{"admin"}, controllers.AlertRulesHandler)))
	mux.HandleFunc("/api/alert-rules/", controllers.RequireRole([]string{"admin"}, controllers.AlertRuleDetailHandler))
	mux.HandleFunc("/api/alert-suppressions", controllers.Audit("manage", "alert_suppression", controllers.RequireRole([]string{"admin"}, controllers.AlertSuppressionsHandler)))
	mux.HandleFunc("/api/alert-suppressions/", controllers.RequireRole([]string{"admin"}, controllers.AlertSuppressionDetailHandler))
	mux.HandleFunc("/api/alert-silences", controllers.Audit("manage", "alert_silence", controllers.RequireRole([]string{"admin"}, controllers.AlertSilencesHandler)))
	mux.HandleFunc("/api/alert-silences/", controllers.RequireRole([]string{"admin"}, controllers.AlertSilenceDetailHandler))
	mux.HandleFunc("/api/alert-escalations", controllers.Audit("manage", "alert_escalation", controllers.RequireRole([]string{"admin"}, controllers.AlertEscalationsHandler)))
	mux.HandleFunc("/api/alert-escalations/", controllers.RequireRole([]string{"admin"}, controllers.AlertEscalationDetailHandler))
	mux.HandleFunc("/api/alert-stats", controllers.RequireRole([]string{"admin", "member"}, controllers.AlertStatsHandler))
	mux.HandleFunc("/api/maintenance/flush-caches", controllers.RequireRole([]string{"admin"}, controllers.FlushCachesHandler))

	// 租户和用户管理
	mux.HandleFunc("/api/tenants", controllers.Audit("manage", "tenant", controllers.RequireRole([]string{"admin"}, controllers.TenantsHandler)))
	mux.HandleFunc("/api/users", controllers.UsersHandler)
	mux.HandleFunc("/api/users/", controllers.Audit("manage", "user", controllers.UserDetailHandler))

	// 系统维护
	mux.HandleFunc("/api/maintenance/downsample", controllers.RequireRole([]string{"admin"}, controllers.DownsampleHandler))

	// Agent Token 管理
	mux.HandleFunc("/api/agent-tokens", controllers.Audit("manage", "agent_token", controllers.RequireRole([]string{"admin"}, controllers.AgentTokensHandler)))
	mux.HandleFunc("/api/agent-tokens/", controllers.RequireRole([]string{"admin"}, controllers.AgentTokenDetailHandler))

	// 审计日志
	mux.HandleFunc("/api/audit-logs", controllers.RequireRole([]string{"admin"}, controllers.AuditLogsHandler))

	// 系统设置
	mux.HandleFunc("/api/settings", controllers.Audit("manage", "setting", controllers.RequireRole([]string{"admin"}, controllers.SettingsHandler)))

	// 健康检查
	mux.HandleFunc("/api/health", controllers.HealthHandler)

	// AI 能力接口
	mux.HandleFunc("/api/llm/settings", controllers.RequireRole([]string{"admin"}, controllers.LLMSettingsHandler))
	mux.HandleFunc("/api/ai/chat", controllers.AIChatHandler)
	mux.HandleFunc("/api/ai/analyze-alert/", controllers.AIAnalyzeHandler)
	mux.HandleFunc("/api/ai/generate-report", controllers.AIGenerateReportHandler)

	// MCP 工具接口
	mux.HandleFunc("/api/mcp/tools", controllers.RequireRole([]string{"admin"}, controllers.MCPToolsHandler))
	mux.HandleFunc("/api/mcp/propose", controllers.MCPProposeHandler)
	mux.HandleFunc("/api/mcp/approvals", controllers.MCPApprovalsHandler)
	mux.HandleFunc("/api/mcp/approvals/", controllers.MCPApprovalActionHandler)

	// 集群管理接口
	mux.HandleFunc("/api/cluster/info", func(w http.ResponseWriter, r *http.Request) {
		controllers.ClusterInfoHandler(w, r, clusterState)
	})
	mux.HandleFunc("/api/cluster/ping", func(w http.ResponseWriter, r *http.Request) {
		controllers.ClusterPingHandler(w, r, clusterState)
	})
	mux.HandleFunc("/api/cluster/leader", func(w http.ResponseWriter, r *http.Request) {
		controllers.ClusterLeaderHandler(w, r, clusterState)
	})

	// /metrics 端点：Prometheus 抓取服务器自监控指标（无需认证）
	mux.Handle("/metrics", http.HandlerFunc(metrics.Handler))

	// 应用中间件（JWT 认证 + 多租户）
	handler := controllers.CORSMiddleware(
		requestCounterMiddleware(controllers.AuthMiddleware(mux)),
	)

	// 定期清理过期的登录失败记录（每 5 分钟）
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			controllers.CleanExpiredAttempts(600)
		}
	}()

	// 定期刷新资源计数（每 30 秒）
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			var hostCount, agentOnline, metricCount int64
			db.DB.Model(&models.Host{}).Count(&hostCount)
			db.DB.Model(&models.Host{}).Where("status = ?", "online").Count(&agentOnline)
			db.DB.Model(&models.MetricSample{}).Count(&metricCount)
			metrics.Global.HostCount.Store(hostCount)
			metrics.Global.AgentCount.Store(agentOnline)
			metrics.Global.MetricCount.Store(metricCount)
		}
	}()

	// 优雅退出
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down server...")
		os.Exit(0)
	}()

	log.Printf("MSMP Server listening on %s", cfg.Server.Addr)
	if err := http.ListenAndServe(cfg.Server.Addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// requestCounterMiddleware 包装内层 handler，统计请求数与错误数。
func requestCounterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		metrics.Global.ObserveRequest(rec.status >= 400)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
