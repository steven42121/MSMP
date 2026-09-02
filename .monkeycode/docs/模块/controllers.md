# controllers — HTTP 处理层

本目录包含所有 HTTP handler，负责请求解析、业务委托、响应序列化。

## 结构

```
controllers/
├── auth.go            # 登录、token 刷新
├── middleware.go      # JWT 认证、租户隔离、Audit 审计、RequireRole 权限
├── hosts.go           # 主机 CRUD、子资源（metrics/tags/events/assets/channels）
├── channels.go        # 渠道 CRUD、probe、ssh-keypair、CollectorScheduler
├── metrics.go         # 监控数据查询
├── alerts.go          # 告警列表、evaluateMetricAlerts
├── alert_rules.go     # 告警规则 CRUD
├── agent_assets.go    # Agent 注册/心跳/资产/指标上报
├── agent_tokens.go    # Agent Token 管理
├── tasks.go           # 远程任务管理
├── tenants.go         # 租户管理
├── users.go           # 用户管理
├── audit.go           # 审计日志查询
└── settings.go        # 系统设置
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `middleware.go` | 核心中间件：CORSMiddleware、AuthMiddleware（JWT 解析+租户注入）、Audit（写操作审计）、RequireRole（角色检查） |
| `channels.go` | 新增：渠道 API + CollectorScheduler（无 Agent 采集调度器） |
| `agent_assets.go` | Agent 上报处理：注册、心跳、资产、指标写入 MetricSample |
| `alerts.go` | 告警评估：evaluateMetricAlerts 每条新样本触发规则匹配 |

## 设计要点

- **路由注册**: 全部在 `server/main.go` 的 `mux.HandleFunc` 中注册
- **租户隔离**: 所有写操作通过 `getTenantID(r)` 获取租户 ID 并附加 WHERE 条件
- **审计日志**: `Audit(action, resource, next)` 包装写操作，记录操作者/资源/状态码
- **响应格式**: 统一使用 `writeJSON(w, status, data)` 输出 JSON

## 依赖

**本模块依赖**:
- `server/models/` — 全部数据模型
- `server/db/` — 数据库连接
- `server/config/` — 配置访问
- `server/collectors/` — 渠道采集（channels.go）
- `server/services/` — 凭据服务（channels.go）
- `github.com/golang-jwt/jwt/v5` — JWT 签名/验证

**依赖本模块的**:
- `server/main.go` — 路由注册和中间件挂载

## 规范

### 新增 Handler

1. 在对应文件末尾添加 handler 函数（接收 `http.ResponseWriter, *http.Request`）
2. 在 `server/main.go` 的 `mux.HandleFunc` 中注册路由
3. 需要鉴权的接口用 `controllers.RequireRole([]string{"admin"}, handler)` 或 `controllers.Audit(action, resource, handler)` 包装
4. 更新 `docs/INTERFACES.md`

### 错误响应

```go
writeJSON(w, http.StatusBadRequest, map[string]string{"error": "描述"})
```
