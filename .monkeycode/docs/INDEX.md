# MSMP 文档索引

MSMP（Mix System Manage Platform）是一个跨平台服务器运维管理平台，支持统一管理 Windows、Linux、macOS、ESXi 等多类主机。平台通过 Agent 采集或远程渠道（SSH、Windows Admin Center、宝塔面板）获取主机的性能指标与资产信息，提供监控可视化、告警联动、任务下发、审计日志等核心能力。

本文档面向三类读者：想了解系统架构的技术负责人、需要集成 API 的开发者、以及准备贡献代码的工程师。

**快速链接**: [架构](./ARCHITECTURE.md) | [接口](./INTERFACES.md) | [开发者指南](./DEVELOPER_GUIDE.md)

---

## 核心文档

### [架构](./ARCHITECTURE.md)
系统架构设计、技术栈、子系统划分与数据流向。了解系统全貌的起点。

### [接口](./INTERFACES.md)
全部 HTTP API 端点、请求/响应格式、认证方式、数据模型定义。外部集成参考。

### [开发者指南](./DEVELOPER_GUIDE.md)
环境搭建、运行命令、编码规范、常见开发任务流程。贡献者必读。

---

## 模块

| 模块 | 描述 | README |
|------|------|--------|
| `server/collectors/` | 无 Agent 采集渠道实现（SSH / WAC / 宝塔面板） | [README](../server/collectors/README.md) |
| `server/services/` | 共享服务（凭据加密等） | [README](../server/services/README.md) |
| `server/controllers/` | HTTP 层，处理所有 REST 请求 | [README](../server/controllers/README.md) |
| `server/models/` | GORM 数据模型与数据库 schema | [README](../server/models/README.md) |
| `server/db/` | 数据库连接与自动迁移 | [README](../server/db/README.md) |
| `agent/` | 部署在目标主机上的采集 Agent | [README](../agent/README.md) |
| `frontend/` | React + Ant Design 前端应用 | [README](../frontend/README.md) |

---

## 核心概念

理解这些领域概念有助于导航代码库：

| 概念 | 描述 |
|------|------|
| [Host](./专有概念/Host.md) | 被管理的主机实体，含状态、资产与监控数据 |
| [MetricSample](./专有概念/MetricSample.md) | 单个时间点的主机性能指标快照 |
| [ChannelBinding](./专有概念/ChannelBinding.md) | 无 Agent 采集渠道与主机的绑定配置 |
| [AlertRule](./专有概念/AlertRule.md) | 基于指标阈值的告警规则 |
| [AgentToken](./专有概念/AgentToken.md) | Agent 注册与认证的凭据 |
| [Tenant](./专有概念/Tenant.md) | 多租户隔离的最小单位 |

---

## 入门指南

### 项目新人？
按此路径学习：
1. **[架构](./ARCHITECTURE.md)** — 了解全局
2. **[核心概念](#核心概念)** — 学习领域术语
3. **[开发者指南](./DEVELOPER_GUIDE.md)** — 搭建环境
4. **[接口](./INTERFACES.md)** — 探索公开 API

### 需要集成？
1. **[接口](./INTERFACES.md)** — API 契约与认证
2. **[架构](./ARCHITECTURE.md)** — 系统边界与数据流

### 首次贡献？
1. **[开发者指南](./DEVELOPER_GUIDE.md)** — 搭建与工作流
2. **[编码规范](./DEVELOPER_GUIDE.md#编码规范)** — 低风险起步点
3. **[常见任务](./DEVELOPER_GUIDE.md#常见任务)** — 分步指南

---

## 快速参考

### 命令

```bash
# 后端
cd server && go run .                       # 开发模式
cd server && go build ./...                  # 编译验证
cd server && go vet ./...                    # 静态检查

# 前端
cd frontend && npm run dev                   # 开发服务器 (port 5173)
cd frontend && npm run build                 # 生产构建
cd frontend && npm run preview               # 本地预览构建产物

# 整体
cd /workspace && go build ./...              # 全项目编译（需 GOTOOLCHAIN=auto）
```

### 配置文件

| 文件 | 用途 |
|------|------|
| `server/config.yaml` | 后端配置（端口、数据库、JWT、Agent 参数、凭据密钥） |
| `frontend/vite.config.js` | 前端开发服务器与反向代理配置 |
| `go.work` | 多模块工作区定义 |

### 重要文件

| 文件 | 目的 |
|------|------|
| `server/main.go` | 后端入口，路由注册，启动调度器 |
| `frontend/src/App.jsx` | 前端路由与认证守卫 |
| `server/db/db.go` | 数据库初始化与 AutoMigrate |
| `server/models/models.go` | 全部数据模型定义 |
| `agent/main.go` | Agent 入口，MainLoop 调度 |
