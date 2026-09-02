package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"MSMP/server/clustering"
	"MSMP/server/config"
	"MSMP/server/controllers"
	"MSMP/server/db"
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

	// 初始化集群状态
	clusterState := clustering.NewClusterState(cfg)

	// 启动离线检测
	go controllers.StartOfflineChecker(cfg.Agent.OfflineAfterSec)

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

	// 租户和用户管理
	mux.HandleFunc("/api/tenants", controllers.Audit("manage", "tenant", controllers.RequireRole([]string{"admin"}, controllers.TenantsHandler)))
	mux.HandleFunc("/api/users", controllers.UsersHandler)
	mux.HandleFunc("/api/users/", controllers.Audit("manage", "user", controllers.UserDetailHandler))

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

	// 应用中间件（JWT 认证 + 多租户）
	handler := controllers.CORSMiddleware(
		controllers.AuthMiddleware(mux),
	)

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
