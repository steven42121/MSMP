# 需求文档：多节点部署与自动负载均衡

Feature Name: multi-node-load-balancing
Updated: 2026-09-02

---

## 概述

MSMP 当前为单节点架构（SQLite + 单进程）。本功能将平台升级为多节点集群，支持水平扩展，并通过客户端侧自动负载均衡实现高可用。Agent 和前端可配置多个后端节点地址，按健康状态自动路由请求；服务端节点间通过心跳互相感知，选举主节点执行共享定时任务（CollectorScheduler、离线检测），避免重复执行。

---

## 术语表

- **节点（Node）**：同一 MSMP 服务实例，运行相同代码，连接同一数据库
- **集群（Cluster）**：一组互相感知的节点集合
- **主节点（Leader）**：集群中负责执行共享定时任务的节点，其他节点为从节点
- **路由（Routing）**：客户端（Agent/前端）决定将请求发往哪个节点的过程
- **健康检查（Health Check）**：节点间或外部 LB 对服务端存活状态的周期性探测

---

## 需求

### 需求 1 — 多节点配置

**用户故事：** AS 运维管理员，我希望在 config.yaml 中配置多个后端节点地址，使 Agent 和前端可以轮询所有可用节点。

#### 验收标准

1. WHEN 启动服务时，IF `server.nodes` 配置项为空，服务端使用当前本机地址加入单节点集群（向后兼容）
2. WHEN 启动服务时，IF `server.nodes` 包含至少两个地址，服务端初始化集群模式并在日志中打印节点列表
3. WHEN 请求 `/api/cluster/info`，IF 集群已初始化，响应返回 `{ "node_id": "...", "mode": "cluster", "leader": "...", "nodes": [...] }`
4. WHILE 服务运行中，IF 收到 SIGTERM，服务端向集群广播 shutdown 信号（最多等待 5 秒）后退出

---

### 需求 2 — 节点发现与心跳

**用户故事：** AS 系统，我希望节点之间自动互相发现并维护心跳，这样集群能感知节点存活状态。

#### 验收标准

1. WHEN 节点启动，IF 配置了 `server.nodes`，节点向所有已知地址发送 HTTP POST `/api/cluster/ping` 心跳请求
2. WHILE 节点运行中，节点每 10 秒向所有已知活跃节点发送心跳，携带本机地址和启动时间
3. IF 连续 3 次心跳失败（间隔 30 秒），节点将目标标记为 `unreachable` 并从活跃列表中移除
4. IF 原 `unreachable` 节点重新发送成功心跳，节点将其恢复为 `reachable`

---

### 需求 3 — Leader 选举

**用户故事：** AS 调度任务，我希望只有一个节点执行 CollectorScheduler 和离线检测，避免指标重复写入。

#### 验收标准

1. WHEN 集群中有 2 个及以上节点，IF 所有节点通过基于地址的字典序比较选举 leader（地址字典序最小者为 leader）
2. WHILE 节点是 leader，leader 执行 CollectorScheduler 和离线检测（`StartOfflineChecker`）
3. WHILE 节点是 follower，跳过 CollectorScheduler 和离线检测的启动
4. IF leader 节点宕机超过 2 个心跳周期（60 秒），follower 重新触发选举，选出新的 leader
5. WHEN 新 leader 上任，原 leader 的定时任务停止，新 leader 立即启动

---

### 需求 4 — Agent 多节点路由

**用户故事：** AS Agent 进程，我希望上报地址支持多个后端节点，自动故障转移，这样当某个节点宕机时我不中断工作。

#### 验收标准

1. WHEN Agent 配置项 `agent.server_urls` 包含多个地址，Agent 维护一个活跃节点轮询列表
2. WHILE Agent 上报指标，IF 当前节点无响应（超时 5 秒），Agent 切换到列表中的下一个节点重试
3. WHEN Agent 连续上报成功 N 次（N=3），Agent 标记该节点为 `healthy`
4. WHEN Agent 连续上报失败 M 次（M=3），Agent 将该节点标记为 `unhealthy` 并跳过该节点
5. IF 所有节点均不可达，Agent 进入指数退避重试（间隔 5s → 10s → 30s，最大间隔 60s）

---

### 需求 5 — 前端多节点路由

**用户故事：** AS 前端用户，我希望前端 API 调用可以自动切换节点，这样当某个后端节点宕机时页面不卡死。

#### 验收标准

1. WHEN 前端初始化时，IF `VITE_MSMP_SERVER_URLS` 环境变量配置多个逗号分隔地址，前端维护可用节点列表
2. WHILE 前端发起 API 请求，IF 当前节点返回非 2xx 或网络超时，前端切换到下一个节点重试一次
3. WHEN 请求成功后，IF 该节点之前标记为不可用，前端将其恢复为可用
4. WHEN 请求失败，IF 该节点连续失败 3 次，前端将其暂时禁用（有效期 60 秒），然后尝试下一个节点
5. IF 所有节点均失败，前端显示统一错误提示"后端服务暂时不可用，请稍后重试"

---

### 需求 6 — 配置示例与文档

**用户故事：** AS 部署工程师，我希望获得清晰的多节点部署配置示例，这样我能快速搭建集群。

#### 验收标准

1. WHEN 部署文档更新，IF 包含双节点和单节点的完整 `config.yaml` 示例
2. WHEN 文档更新，IF 包含 Agent 多节点配置的示例（`agent.server_urls`）
3. WHEN 文档更新，IF 包含 Docker Compose 示例（可选，用于本地测试）
4. WHEN 代码提交，IF `go build ./...` 通过，且无新增编译警告

---

## 不影响范围

- 不改变现有 Agent 单节点配置格式（`agent.server_url` 保持向后兼容，解析为单元素数组）
- 不引入新的数据库迁移（仅切换存储驱动）
- 不修改采集渠道（SSH/WAC/BaoTa/Prometheus/SNMP/WinRM）逻辑
- 不引入额外外部依赖（节点发现使用 HTTP，不依赖 Consul/Etcd）
