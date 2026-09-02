# User Instruction Memory

This file records user instructions, preferences, and teachings for reference in future interactions.

## Format

### User Instruction Entry
User instruction entries should follow this format:

[User Instruction Summary]
- Date: [YYYY-MM-DD]
- Context: [Mentioned scenario or time]
- Instructions:
  - [Content of user teaching or instruction, described line by line]

### Project Knowledge Entry
Entries discovered by the Agent during task execution should follow this format:

[Project Knowledge Summary]
- Date: [YYYY-MM-DD]
- Context: Discovered by Agent while performing [specific task description]
- Category: [Operations & Deployment|Build Methods|Testing Methods|Troubleshooting & Debugging|Workflow & Collaboration|Environment Configuration]
- Instructions:
  - [Specific knowledge points, described line by line]

## Deduplication Strategy
- Before adding a new entry, check for similar or identical instructions.
- If a duplicate is found, skip the new entry or merge it with the existing one.
- When merging, update the context or date information.
- This helps avoid redundant entries and keeps the memory file tidy.

## Entries

[MSMP 构建与运行命令]
- Date: 2026-08-23
- Context: Discovered by Agent while building and verifying the MSMP project
- Category: Build Methods
- Instructions:
  - 前端构建：`cd /workspace/frontend && npm install && npm run build`
  - 后端构建：`cd /workspace/server && go build ./...`
  - Agent 默认仅支持 Windows（`agent/win/collect.go` 使用 `syscall.NewLazyDLL`），在 Linux 上构建需交叉编译：`cd /workspace/agent && GOOS=windows go build ./...`
  - 本地实际 Go 工具链为 1.25.6，而 `go.work`/`go.mod` 要求 1.26.1，需保留 `GOTOOLCHAIN=auto` 让其自动下载工具链；设置 `GOTOOLCHAIN=local` 会导致构建失败
  - 前端开发服务器端口 5173，已配置 `/api` 反向代理到后端 `http://localhost:8080`，并已将 `.monkeycode-ai.online` 加入 `allowedHosts`

[Agentless Metrics Channels 部署要点]
- Date: 2026-09-01
- Context: 实现无 Agent 多渠道采集特性（SSH/WAC/宝塔面板）后的部署与验证
- Category: Environment Configuration
- Instructions:
  - 必须配置 security.credentialkey（base64 32字节 AES 密钥），否则创建渠道绑定返回 503；生成方式：python3 -c "import os,base64;print(base64.b64encode(os.urandom(32)).decode())"
  - 可通过环境变量 MSMP_SECURITY_CREDENTIALKEY 注入（与 viper 的 security.credentialkey 对应）
  - JWT secret 默认为 change-me-in-production，测试签发的 admin token 可用于直接验证 admin API
  - 新渠道 API 路径：GET/POST /api/hosts/{uuid}/channels，PUT/DELETE /api/channels/{id}，POST /api/channels/ssh-keypair（需 admin 角色）
  - 前端「采集渠道」tab 位于 HostDetail 页面，需配合后端运行才能看到

[无 Agent 多渠道采集特性实施]
- Date: 2026-09-01
- Context: 为 MSMP 主机性能监控增加 SSH / Windows Admin Center / 宝塔面板三种无 Agent 采集渠道
- Category: Build Methods & Operations
- Instructions:
  - 新建代码位置：server/collectors/（Channel 接口 + 三个渠道实现）、server/services/credential.go（AES-256-GCM 凭据加解密 + SSH 密钥对生成）、server/controllers/channels.go（渠道 API + CollectorScheduler）
  - 调度器行为：60 秒周期、Agent 优先跳过（5 分钟内 Agent 有上报则不触发渠道）、按 Priority 优先级降级、单渠道连续失败 5 次自动禁用
  - SSH 渠道在 Linux 上通过 /proc 文件解析指标，语义与 Agent 端 MetricData 完全一致；首次成功采集会同步更新 host 资产字段
  - 所有凭据（密码/私钥/API Key）均经 AES-256-GCM 加密落库，任何 API 响应绝不返回明文
  - 前端「采集渠道」tab 位于 HostDetail 页面（/hosts/:uuid），含创建向导（SSH 支持密码/自建私钥/平台生成密钥对三种接入方式）
  - 后端已验证的 API：POST /api/channels/ssh-keypair（返回公钥 + 一键安装命令 + 加密私钥）、POST /api/hosts/{uuid}/channels（创建并自动探测）、POST /api/channels/{id}/probe（手动探测）
  - 端到端测试路径：手动添加主机 → 绑定 SSH 渠道 → 探测成功后等待 ≤60s 调度采集 → 前端监控曲线出现数据 → 告警规则自动触发（复用 evaluateMetricAlerts）

[多节点部署与负载均衡]
- Date: 2026-09-02
- Context: 将 MSMP 从单节点升级为多节点集群，实现 Agent 和前端自动故障转移
- Category: Operations & Deployment
- Instructions:
  - 集群状态管理在 server/clustering/cluster.go，Leader 选举基于节点地址字典序（最小者胜出），follower 跳过 CollectorScheduler
  - 节点间心跳 POST /api/cluster/ping（每 10s），连续 3 次失败（30s）标记 unreachable
  - 配置方式：config.yaml 中 server.nodes 数组填入所有节点地址，server.node_id 留空自动生成
  - Agent 多节点路由：MSMP_SERVER_URLS 环境变量逗号分隔多个地址，连续失败 3 次跳过该节点
  - 前端多节点路由：VITE_MSMP_SERVER_URLS 环境变量逗号分隔，失败 3 次后禁用 60 秒
  - 新 API 端点：GET /api/cluster/info、POST /api/cluster/ping、GET /api/cluster/leader


[Project Knowledge Summary]
- Date: 2026-09-02
- Context: Added LLM auto-management with MCP operation capabilities
- Category: Build Methods
- Instructions:
  - 后端新增 MCP 工具系统：server/services/mcp_tools.go, server/services/mcp_service.go, server/controllers/mcp.go
  - 可用工具：list_hosts, get_host_status, get_recent_alerts, execute_command, check_service, view_logs, generate_report
  - API 端点：GET /api/mcp/tools, POST /api/mcp/propose, GET /api/mcp/approvals, POST /api/mcp/approvals/:id/approve, POST /api/mcp/approvals/:id/reject
  - AI Chat 页面支持工具提案和审批 UI：frontend/src/pages/AIChat.jsx
  - 危险操作（execute_command, check_service, view_logs）需要用户审批才能执行
  - 安全操作（list_hosts, get_host_status, get_recent_alerts, generate_report）可直接执行

[Project Knowledge Summary]
- Date: 2026-09-02
- Context: Implemented fluid liquid glass effect with mouse tracking
- Category: Build Methods
- Instructions:
  - 液态玻璃效果通过 CSS 变量 --glow-x/--glow-y 动态控制高光位置
  - useGlobalMouseTracker hook 在 document 级别跟踪鼠标，更新 :root 变量
  - 所有 .liquid-glass 元素自动响应鼠标移动
  - glass-fluid 层使用 conic-gradient 旋转动画模拟液体光线折射
  - Dark mode 高光带蓝色氛围光 (rgba(90, 130, 255))
  - LiquidGlass 组件提供 per-card 鼠标跟踪（可选）
