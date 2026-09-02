# 实施计划：多节点部署与自动负载均衡

## 阶段一：服务端集群基础

- [ ] 1.1 新建 `server/clustering/cluster.go`
  - 实现 `ClusterState` 结构体（内存维护节点列表）
  - 实现 `IsLeader()`：基于地址字典序比较
  - 实现 `RegisterNode(address, nodeID)`：upsert 语义
  - 实现 `DeregisterDeadNodes(threshold)`：超时清理
  - 实现 `GetHealthyNodes() []string`
  - 实现 `StartHeartbeatLoop()`：follower 定时向所有节点发 ping

- [ ] 1.2 新建 `server/controllers/cluster.go`
  - `GET /api/cluster/info`：返回 `{ node_id, mode, leader, nodes[] }`
  - `POST /api/cluster/ping`：接收心跳，更新状态，返回当前 leader
  - `GET /api/cluster/leader`：返回当前 leader 地址
  - 引用：design.md「组件设计 2」

- [ ] 1.3 修改 `server/config/config.go`
  - `ServerConfig` 新增 `Nodes []string` 和 `NodeID string`
  - 添加默认值：`server.nodes` 空列表（单节点模式），`server.node_id` 自动生成 UUID
  - 引用：design.md「组件设计 4」

- [ ] 1.4 修改 `server/main.go`
  - 初始化 `clustering.ClusterState`
  - IF leader：启动 `StartOfflineChecker` 和 `StartCollectorScheduler`
  - IF follower：跳过上述启动，改为 `clusterState.JoinCluster()`
  - 注册 `/api/cluster/*` 路由
  - 引用：design.md「组件设计 5」

- [ ] 1.5 更新 `server/config.yaml`
  - 添加双节点示例配置块
  - 添加 `server.nodes` 和 `server.node_id` 字段说明
  - 引用：requirements.md Req 6

---

## 阶段二：客户端路由

- [ ] 2.1 修改 `frontend/src/api/client.js`
  - 将 axios 实例封装为 `ClusterClient`
  - 实现节点健康跟踪（成功计数 / 失败计数 / 禁用窗口）
  - 实现轮询路由：`request(method, path, options)` 按顺序尝试所有健康节点
  - 从 `import.meta.env.VITE_MSMP_SERVER_URLS` 读取节点列表（逗号分隔）
  - 引用：design.md「组件设计 3 - 前端」

- [ ] 2.2 修改 `frontend/vite.config.js`
  - 添加 `VITE_MSMP_SERVER_URLS` 环境变量读取
  - 开发代理支持多目标（按轮询分配）
  - 引用：design.md「组件设计 3」

- [ ] 2.3 修改 `agent/common/agent.go`
  - 新增 `ClusterRouter` 结构体
  - 实现 `NextNode()` 轮询逻辑
  - 实现失败切换：连续失败 3 次标记 unhealthy，切换下一节点
  - 引用：design.md「组件设计 3 - Agent」

- [ ] 2.4 修改 `agent/main.go`
  - 读取 `MSMP_SERVER_URLS`（逗号分隔，兼容单值）
  - 初始化 `ClusterRouter` 并传入 `MainLoop`
  - 上报时调用 `router.NextNode()` 获取目标 URL
  - 引用：requirements.md Req 4

---

## 阶段三：验证与文档

- [ ] 3.1 编写 `docs/deploy-multi-node.md`
  - 单节点配置示例（现有配置保持不变）
  - 双节点配置示例（config.yaml + Docker Compose）
  - Agent 多节点配置说明
  - Leader 选举机制说明
  - 故障转移行为说明
  - 引用：requirements.md Req 6

- [ ] 3.2 本地集成验证
  - 启动两个服务器实例（不同端口）
  - 验证 `/api/cluster/info` 返回两个节点均 alive
  - 验证其中一个停止后，另一个成为 leader
  - 验证 Agent 配置多节点地址后自动故障转移
  - 引用：requirements.md Req 2、3、4、5

- [ ] 3.3 更新项目文档
  - 更新 `docs/ARCHITECTURE.md`：添加集群架构图和组件依赖图
  - 更新 `docs/INTERFACES.md`：新增集群相关 API 端点
  - 更新 `docs/DEVELOPER_GUIDE.md`：新增多节点启动步骤
  - 引用：requirements.md Req 6

---

## 检查点

- [ ] CP1：服务端集群基础完成（阶段一全部提交）
- [ ] CP2：Agent 多节点路由完成（阶段二 2.3+2.4 提交）
- [ ] CP3：前端多节点路由完成（阶段二 2.1+2.2 提交）
- [ ] CP4：端到端验证完成（阶段三 3.2 通过）
