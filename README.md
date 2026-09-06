# MSMP — Mix System Manage Platform

跨平台服务器运维管理平台，统一管理 Windows、Windows Server、Linux、macOS、ESXi 等主机。

<p align="center">
  <b>监控可视化</b> · <b>智能告警</b> · <b>远程终端</b> · <b>文件管理</b> · <b>虚拟化采集</b> · <b>AI 运维</b>
</p>

## 功能特性

| 能力 | 说明 |
|------|------|
| **主机监控** | Agent 采集 + 无 Agent 多渠道采集（SSH / Windows Admin Center / 宝塔面板 / Prometheus / SNMP / WinRM / vSphere / Proxmox VE），CPU、内存、磁盘、网络、进程、负载、温度、GPU 等指标 |
| **智能告警** | 阈值告警规则 + 抑制（防风暴）+ 静默（维护窗口）+ 升级（未处理超时自动升级），Webhook 通知 |
| **远程终端** | 浏览器内 WebSSH，WebSocket 直通目标主机 SSH，支持终端尺寸自适应 |
| **文件管理** | SFTP 远程文件浏览、上传、下载、删除、重命名、新建目录 |
| **虚拟化管理** | ESXi/vCenter vSphere API + Proxmox VE REST API：采集节点与虚拟机指标，虚拟机/容器电源操作（开机/关机/重启/挂起），数据存储容量监控 |
| **AI 运维** | MCP 工具体系，AI 可查主机状态、执行命令（危险操作人工审批）、分析告警根因 |
| **安全** | 多租户隔离、角色权限（admin/member）、AES-256-GCM 凭据加密、完整审计日志、登录失败锁定、IP 白名单 |

## 技术栈

- **后端**：Go 1.26，标准库 `net/http` + GORM（SQLite / PostgreSQL）
- **前端**：React 18 + Vite 5 + Ant Design 5 + Zustand + ECharts
- **Agent**：Go 编译的独立二进制，gopsutil 采集
- **远程协议**：SSH / SFTP / WinRM / SNMP / vSphere（govmomi）/ Proxmox VE（REST API）

## 架构

```
┌─────────────┐     REST/WebSocket      ┌──────────────────┐
│   前端 React │ ──────────────────────▶ │   后端 Go Server  │
│  (Vite 5173) │ ◀────────────────────── │    (HTTP :8080)   │
└─────────────┘                         └────────┬─────────┘
                                                  │ 心跳 / 指标上报
                                      ┌───────────▼───────────┐
                                      │    Agent（目标主机）    │
                                      │  (CPU/内存/磁盘/进程...) │
                                      └───────────────────────┘
                                                  │
                     ┌────────────────────────────┼────────────────┐
                     ▼                            ▼                ▼
               SSH / SFTP / WinRM            vSphere API       SNMP / 宝塔
              （无 Agent 远程采集）          （ESXi / vCenter）  （网络设备）
```

## 快速开始

### 前置要求

- Go 1.25+（自动下载工具链）
- Node.js 18+
- SQLite（默认，无需额外配置）

### 1. 启动后端

```bash
cd server
go run main.go
```

后端默认监听 `:8080`，配置见 `server/config.yaml`。

首次启动会自动创建 `msmp.db` 数据库并完成表迁移。

### 2. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端监听 `:5173`，已配置 `/api` 反向代理到后端 `:8080`。访问 http://localhost:5173。

### 3. 部署 Agent

```bash
cd agent
go build -o msmp-agent .
MSMP_SERVER_URL=http://your-server:8080 AGENT_UUID=host-001 ./msmp-agent
```

Agent 环境变量：

| 变量 | 说明 |
|------|------|
| `MSMP_SERVER_URL` | 服务端地址（可逗号分隔多节点） |
| `AGENT_UUID` | 主机标识，留空则用 hostname |
| `AGENT_TOKEN` | Agent 接入凭证（可选） |

## 构建

```bash
# 后端
cd server && go build -o msmp-server .

# 前端（产物在 frontend/dist/）
cd frontend && npm run build

# Agent（当前平台）
cd agent && go build -o msmp-agent .

# Agent（交叉编译 Windows）
cd agent && GOOS=windows GOARCH=amd64 go build -o msmp-agent.exe .
```

## Docker 部署

```bash
docker compose up -d
```

见 `docker-compose.yml`，包含后端与前端。

## 配置

核心配置项位于 `server/config.yaml`：

| 配置 | 说明 |
|------|------|
| `server.addr` | 后端监听地址，默认 `:8080` |
| `db.driver` | `sqlite` 或 `postgres` |
| `jwt.secret` | JWT 签名密钥，**生产环境务必修改** |
| `security.credentialkey` | 32 字节 base64 密钥，用于加密采集渠道凭据 |

生成凭据密钥：

```bash
python3 -c "import os,base64;print(base64.b64encode(os.urandom(32)).decode())"
```

## 项目文档

- 架构设计：[.monkeycode/docs/ARCHITECTURE.md](.monkeycode/docs/ARCHITECTURE.md)
- 接口文档：[.monkeycode/docs/INTERFACES.md](.monkeycode/docs/INTERFACES.md)
- 开发者指南：[.monkeycode/docs/DEVELOPER_GUIDE.md](.monkeycode/docs/DEVELOPER_GUIDE.md)

## Roadmap

### 已完成
- [x] 多源主机监控（Agent + 7 种无 Agent 渠道）
- [x] 阈值告警 + 抑制/静默/升级工程化
- [x] WebSSH 远程终端 + SFTP 文件管理
- [x] ESXi/vCenter vSphere API 采集
- [x] MCP AI 工具体系（危险操作人工审批）
- [x] 多租户隔离、角色权限、AES-256-GCM 凭据加密
- [x] Docker 容器化部署 + GitHub Release 自动打包

### 计划中
- [ ] Agent 自动升级（服务端下发版本 → Agent 自拉取）
- [ ] VM 深度管理（开关机、快照创建/恢复）
- [ ] 时序数据降采样 + 保留策略
- [ ] 终端会话审计回放
- [ ] 主机组维度的细粒度权限

## License

[MIT](LICENSE)