package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	// 注册路由
	mux := http.NewServeMux()

	// Agent 相关接口
	mux.HandleFunc("/api/agents/register", controllers.AgentRegisterHandler)
	mux.HandleFunc("/api/agents/heartbeat", controllers.AgentHeartbeatHandler)
	mux.HandleFunc("/api/agents/assets", controllers.AgentAssetReportHandler)
	mux.HandleFunc("/api/agents/metrics", controllers.AgentMetricReportHandler)

	// 认证接口
	mux.HandleFunc("/api/auth/login", controllers.LoginHandler)
	mux.HandleFunc("/api/auth/refresh", controllers.RefreshTokenHandler)

	// 主机管理接口
	mux.HandleFunc("/api/hosts", controllers.HostsHandler)
	mux.HandleFunc("/api/hosts/", controllers.HostDetailHandler)

	// 监控数据接口
	mux.HandleFunc("/api/metrics", controllers.MetricsHandler)

	// 任务管理接口
	mux.HandleFunc("/api/tasks", controllers.TasksHandler)
	mux.HandleFunc("/api/tasks/", controllers.TaskDetailHandler)

	// 事件/告警接口
	mux.HandleFunc("/api/alerts", controllers.AlertsHandler)

	// 租户和用户管理
	mux.HandleFunc("/api/tenants", controllers.TenantsHandler)
	mux.HandleFunc("/api/users", controllers.UsersHandler)

	// 健康检查
	mux.HandleFunc("/api/health", controllers.HealthHandler)

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