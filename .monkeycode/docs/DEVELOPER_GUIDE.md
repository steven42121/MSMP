# 开发者指南

## 项目目的

MSMP 是一个跨平台服务器运维管理平台。它在更大系统中担任**主机运维中枢**的角色，负责采集、存储、展示和告警联动，是运维自动化链路的核心环节。

**核心职责**:
- 管理多租户环境下的主机资产与性能监控
- 支持 Agent 采集与无 Agent 远程采集（SSH/WAC/宝塔）两种模式
- 提供实时监控曲线、告警规则配置与 Webhook 通知
- 记录完整的操作审计日志

**相关系统**:
- MSMP Agent — 部署在目标主机上的独立采集进程
- 目标主机（Linux/Windows/macOS/ESXi）— 数据采集来源

---

## 环境搭建

### 前置条件

- Go 1.25+（开发环境会自动下载 1.26.1 toolchain，设置 `GOTOOLCHAIN=auto`）
- Node.js 18+
- SQLite（内置，无需额外安装）或 PostgreSQL（可选）

### 安装

```bash
# 克隆仓库
git clone <repo-url>
cd MSMP

# 后端
cd server && go mod download

# 前端
cd ../frontend && npm install
```

### 环境变量

| 变量 | 必需 | 描述 | 示例 |
|------|------|------|------|
| `MSMP_SECURITY_CREDENTIALKEY` | 否* | AES-256-GCM 凭据加密密钥（base64 32字节），不配置则无法创建采集渠道 | `bdzB7G...` |
| `MSMP_SERVER_ADDR` | 否 | 监听地址 | `:8080` |
| `MSMP_DB_DRIVER` | 否 | 数据库驱动 | `sqlite` / `postgres` |
| `MSMP_DB_SQLITEPATH` | 否 | SQLite 文件路径 | `msmp.db` |
| `MSMP_DB_DSN` | 否 | PostgreSQL 连接串 | `host=localhost ...` |
| `MSMP_JWT_SECRET` | 否 | JWT 签名密钥 | `your-secret-here` |

\* 仅在使用无 Agent 采集渠道时需要配置。

生成密钥：
```bash
python3 -c "import os,base64;print(base64.b64encode(os.urandom(32)).decode())"
```

⚠️ **绝不提交密钥**。生产环境请通过环境变量注入，勿修改 `config.yaml` 中的实际值。

### 运行

```bash
# 后端（终端 1）
cd server && go run .

# 前端（终端 2）
cd frontend && npm run dev
```

前端开发服务器运行在 `http://localhost:5173`，自动将 `/api/*` 请求反向代理到 `http://localhost:8080`。

### Agent 运行（可选）

```bash
# Linux 上编译并运行
cd agent && GOOS=linux go build -o msmp-agent .
./msmp-agent --server http://localhost:8080 --token <token>

# Windows 上交叉编译
cd agent && GOOS=windows go build -o msmp-agent.exe .
```

---

## 开发工作流

### 代码质量

本项目目前无 lint/formatter 自动化配置。手动检查：
```bash
# 后端编译验证
cd server && go build ./... && go vet ./...

# 前端构建验证
cd frontend && npm run build
```

### 分支策略

- `main` — 生产就绪代码
- `feature/*` — 新功能（如 `feature/agentless-metrics-channels`）
- `fix/*` — Bug 修复

### 提交规范

遵循 Conventional Commits：
```
feat(server): add SSH channel collector
fix(agent): handle empty /proc/stat
docs: update architecture diagram
chore: bump gopsutil to v3
```

---

## 编码规范

### 文件组织

- 后端：每个 handler/service 一个文件，模型集中在 `models/models.go`
- 前端：每个页面一个组件文件，store 按域拆分

### 命名约定

| 类型 | 约定 | 示例 |
|------|------|------|
| 文件（Go） | snake_case | `agent_assets.go` |
| 文件（JSX） | PascalCase | `HostDetail.jsx` |
| 函数/方法（Go） | PascalCase | `AgentRegisterHandler` |
| 函数（JS） | camelCase | `loadMetrics` |
| 常量（Go） | SCREAMING_SNAKE | `StatusUnreachable` |
| 结构体 | PascalCase | `MetricSample` |

### 错误处理

Go 侧：错误返回给调用方，handler 层统一转为 JSON 响应。
JS 侧：使用 try/catch，错误消息通过 Ant Design message 组件展示。

### 日志

```go
log.Printf("[scheduler] collecting host=%d err=%v", hostID, err)
```

---

## 常见任务

### 添加新 API 端点

1. 在 `server/controllers/` 新增或修改 handler 文件
2. 在 `server/main.go` 注册路由
3. 如需新数据模型，在 `server/models/models.go` 添加 struct 并在 `server/db/db.go` AutoMigrate 列表加入
4. 前端如有对应页面，在 `frontend/src/pages/` 新增组件，在 `App.jsx` 注册路由
5. 更新 `docs/INTERFACES.md`

### 添加新采集渠道

1. 在 `server/collectors/` 实现 `Channel` 接口
2. 在 `server/controllers/channels.go` 的 `initCollectors()` 中注册
3. 前端 `HostDetail.jsx` 的创建向导中添加对应选项

### 添加新环境变量

1. 在 `server/config/config.go` 的 Config 结构体添加字段
2. 在 `server/config/config.go` 的 `Load()` 中添加 `v.SetDefault(...)`
3. 在 `server/config.yaml` 中添加示例配置
4. 更新本指南的「环境变量」表格

### 修复 Bug

1. 定位根因代码
2. 编写复现步骤或最小测试用例
3. 最小改动修复
4. 运行 `go build ./...` 验证
5. 检查同类问题是否存在

---

## 重要设计决策

### 依赖注入模式

服务端无 DI 框架，依赖通过全局变量和 init 函数管理：
- `db.DB` — 全局 GORM 实例
- `config.C` — 全局配置指针
- `channelReg` — 渠道注册表（包级变量）
- `credService` — 凭据服务（懒初始化）

### 多租户隔离

所有数据查询必须附加 `WHERE tenant_id = ?`，通过 AuthMiddleware 从 JWT 中提取 tenantID 并注入 context。Controller 通过 `getTenantID(r)` 获取。

### 凭据安全

敏感凭据（SSH 密码/私钥、API Key）经 AES-256-GCM 加密后落库。`ChannelBinding.Credential` 字段的 `json:"-"` 标签确保响应永不返回明文。

### Agent 优先策略

CollectorScheduler 仅在主机 `last_heartbeat` 距今超过 5 分钟时才会尝试无 Agent 采集，确保 Agent 上报优先级最高。
