# MSMP — Mix System Manage Platform

跨平台服务器运维管理平台，统一管理 Windows、Windows Server、Linux、macOS、ESXi、Proxmox VE 等主机。

<p align="center">
  <b>监控可视化</b> · <b>智能告警</b> · <b>远程终端</b> · <b>文件管理</b> · <b>虚拟化采集</b> · <b>插件体系</b> · <b>网络安全管理</b> · <b>多节点集群</b>
</p>

## 功能特性

| 能力 | 说明 |
|------|------|
| **主机监控** | Agent 采集 + 9 种无 Agent 渠道（SSH / Windows Admin Center / 宝塔面板 / 1Panel / Prometheus / SNMP / WinRM / vSphere / Proxmox VE），CPU、内存、磁盘、网络、进程、负载、GPU 等指标 |
| **智能告警** | 阈值规则 + 抑制（防风暴）+ 静默（维护窗口）+ 升级（未处理自动升级）；通知分发到多渠道（Webhook / 钉钉 / 飞书 / 企业微信 / 邮件 SMTP / Slack） |
| **远程终端** | 浏览器内 WebSSH + 会话录制回放；SFTP 文件浏览、上传、下载、重命名 |
| **虚拟化管理** | ESXi vSphere + Proxmox VE：虚拟机/容器电源操作、快照管理、备份、guest 详情、数据存储、网络、集群聚合视图 |
| **插件体系** | 通用插件框架：通知/采集/探测统一纳管，插件实例可配置、启停、测试，敏感字段加密与掩码 |
| **网络安全管理** | 端口暴露风险审计（18 类高风险端口）、安全基线检查、防火墙管理（firewalld/ufw） |
| **多节点集群** | 统一 PostgreSQL 共享库，节点自动选举 leader，前端节点管理，nginx 负载均衡，一键多节点部署 |
| **可用性探测** | HTTP/TCP/SSL 探测 + Cron 定时任务调度器 |
| **可观测性** | Prometheus 格式 `/metrics`、时序降采样与保留策略、进程/端口/软件包资产清单 |
| **Agent 运维** | 自动升级（服务端下发）、多节点故障转移、优雅退出、零停机更新 |
| **安全** | 多租户隔离、角色权限（admin/member）、AES-256-GCM 凭据加密、审计日志、登录锁定、IP 白名单 |

## 技术栈

- **后端**：Go，标准库 `net/http` + GORM（PostgreSQL，SQLite 用于本地快速验证）
- **前端**：React 18 + Vite 5 + Ant Design 5 + Zustand + ECharts
- **Agent**：Go 静态编译的独立二进制（linux / darwin / windows / arm64），gopsutil 采集
- **远程协议**：SSH / SFTP / WinRM / SNMP / vSphere（govmomi）/ Proxmox VE / 1Panel（REST API）

## 架构

```
                    ┌──────────────────────────────┐
                    │         PostgreSQL（共享库）     │
                    └──────────────▲───────────────┘
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
┌───────▼───────┐          ┌───────▼───────┐          ┌───────▼───────┐
│  Server 节点 1 │◀──心跳──▶│  Server 节点 2 │◀──心跳──▶│  Server 节点 N │
│   (leader)    │          │   (follower)   │          │   (follower)   │
└───────▲───────┘          └───────▲───────┘          └───────▲───────┘
        │  REST/WebSocket          │                          │
        └────────────────┬─────────┴──────────────────────────┘
                         │ nginx 负载均衡
                  ┌──────▼───────┐
                  │   前端 React   │
                  └──────────────┘
                         ▲
                         │ 心跳 / 指标 / 资产上报
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
     Agent(Linux)    Agent(macOS)    无 Agent 远程采集
   (CPU/内存/进程...)  (darwin)     (SSH/WinRM/SNMP/vSphere/PVE/1Panel)
```

## 快速开始

### 前置要求

- Go 1.26+（自动下载工具链）
- Node.js 18+
- PostgreSQL（默认存储）或 SQLite（本地快速验证）

### 1. 启动数据库

默认使用 PostgreSQL，先准备数据库：

```bash
# 使用 Docker
docker run -d --name msmp-pg -e POSTGRES_USER=msmp -e POSTGRES_PASSWORD=msmp123 \
  -e POSTGRES_DB=msmp -p 5432:5432 postgres:16-alpine
```

也可以直接用 Docker Compose 一键部署（见下文），或改用 SQLite。

### 2. 启动后端

```bash
cd server
go run main.go
```

后端默认监听 `:8080`，配置见 `server/config.yaml`（默认 `db.driver=postgres`）。首次启动自动建表。

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端监听 `:5173`，`/api` 反向代理到后端 `:8080`。访问 http://localhost:5173。

### 4. 部署 Agent

```bash
cd agent
go build -o msmp-agent .
MSMP_SERVER_URLS=http://your-server:8080 AGENT_UUID=host-001 AGENT_TOKEN=xxx ./msmp-agent
```

Agent 环境变量：

| 变量 | 说明 |
|------|------|
| `MSMP_SERVER_URLS` | 服务端地址（逗号分隔多节点，自动故障转移） |
| `AGENT_UUID` | 主机标识，留空则用 hostname |
| `AGENT_TOKEN` | Agent 接入凭证 |

## Docker 部署

单机：

```bash
docker compose up -d
```

多节点集群（PostgreSQL + 3 个 server 节点 + 前端负载均衡 + Agent）：

```bash
docker compose -f docker-compose.cluster.yml up -d
```

详细说明见 `.monkeycode/docs/CLUSTER_DEPLOY.md`。

## 配置

核心配置项位于 `server/config.yaml`，全部支持 `MSMP_` 前缀环境变量覆盖：

| 配置 | 环境变量 | 说明 |
|------|---------|------|
| `server.addr` | `MSMP_SERVER_ADDR` | 后端监听地址，默认 `:8080` |
| `server.advertise_addr` | `MSMP_SERVER_ADVERTISE_ADDR` | 集群内通告地址（leader 选举用） |
| `server.nodes` | `MSMP_SERVER_NODES` | 集群节点列表（逗号分隔） |
| `server.node_id` | `MSMP_SERVER_NODE_ID` | 当前节点标识 |
| `db.driver` | `MSMP_DB_DRIVER` | `postgres`（默认）或 `sqlite` |
| `db.dsn` | `MSMP_DB_DSN` | PostgreSQL DSN |
| `jwt.secret` | `MSMP_JWT_SECRET` | JWT 签名密钥，**生产环境务必修改** |
| `security.credentialkey` | `MSMP_SECURITY_CREDENTIALKEY` | 32 字节 base64 密钥，加密采集渠道凭据（所有节点需一致） |

生成凭据密钥：

```bash
python3 -c "import os,base64;print(base64.b64encode(os.urandom(32)).decode())"
```

## 构建

```bash
# 后端
cd server && go build -o msmp-server .

# 前端（产物在 frontend/dist/）
cd frontend && npm run build

# Agent 全平台
make agent
```

## 项目文档

- 架构设计：当前工作区 内的 `.monkeycode/docs/ARCHITECTURE.md`
- 接口文档：当前工作区 内的 `.monkeycode/docs/INTERFACES.md`
- 多节点部署：当前工作区 内的 `.monkeycode/docs/CLUSTER_DEPLOY.md`
- 开发者指南：当前工作区 内的 `.monkeycode/docs/DEVELOPER_GUIDE.md`

## Roadmap

### 已完成
- [x] 多源主机监控（Agent + 9 种无 Agent 渠道）
- [x] 阈值告警 + 抑制/静默/升级 + 多渠道通知
- [x] WebSSH + 会话录制回放 + SFTP 文件管理
- [x] ESXi/vSphere + Proxmox VE 虚拟化管理（电源/快照/备份/详情/网络/集群）
- [x] 1Panel 采集渠道
- [x] 通用插件框架（通知/采集/探测）
- [x] 网络安全管理（端口风险 / 基线检查 / 防火墙管理）
- [x] 多节点集群 + PostgreSQL 共享库 + 一键部署
- [x] Prometheus /metrics + 时序降采样 + 保留策略
- [x] 可用性探测 + Cron 调度
- [x] Agent 自动升级 + 零停机更新
- [x] 多租户隔离、角色权限、AES-256-GCM 凭据加密
- [x] Docker 部署 + GitHub Release 多平台打包（linux/darwin/windows/arm64/esxi/pve）

### 计划中
- [ ] VM 创建/克隆/迁移（待真实 PVE/ESXi 环境验证）
- [ ] noVNC / WebMKS 虚拟控制台（待真实环境）
- [ ] 移动端只读查看器（PWA）
- [ ] 采集调度器多节点运行时 leader 动态迁移
- [ ] 主机组维度细粒度权限

## License

[MIT](LICENSE)