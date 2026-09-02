# 多节点部署与自动负载均衡

Feature Name: multi-node-load-balancing
Updated: 2026-09-02

---

## 概述

将 MSMP 从单节点架构升级为支持多节点集群，通过客户端侧健康轮询实现自动故障转移，通过基于地址字典序的简单 Leader 选举保证共享定时任务只由一个节点执行。不引入外部协调服务（Consul/Etcd），所有节点发现通过 HTTP 心跳完成。

---

## 架构

```mermaid
flowchart TD
    subgraph Clients["客户端层"]
        A1[前端 Vite]
        A2[Agent Linux]
        A3[Agent Windows]
    end

    subgraph LB["客户端路由"]
        R1[节点列表轮询]
        R2[健康状态跟踪]
        R3[失败切换 + 退避]
    end

    subgraph Cluster["MSMP 集群（3 节点示例）"]
        N1[Node A :8080\nleader]
        N2[Node B :8081\nfollower]
        N3[Node C :8082\nfollower]
    end

    subgraph Shared["共享存储"]
        DB[(PostgreSQL)]
    end

    subgraph Scheduler["共享任务（仅 leader 执行）"]
        SC1[CollectorScheduler\n60s ticker]
        SC2[OfflineChecker\n30s ticker]
    end

    A1 -->|轮询可用节点| R1
    A2 -->|轮询可用节点| R1
    A3 -->|轮询可用节点| R1
    R1 -->|健康节点| N1
    R1 -->|健康节点| N2
    R1 -->|健康节点| N3
    R2 -->|失败计数| R3
    R3 -->|切换到备用节点| N2
    R3 -->|指数退避| N3

    N1 <-->|心跳 POST /api/cluster/ping| N2
    N1 <-->|心跳 POST /api/cluster/ping| N3
    N2 <-->|心跳 POST /api/cluster/ping| N3

    N1 -->|写入| DB
    N2 -->|写入| DB
    N3 -->|写入| DB

    N1 -.->|执行| SC1
    N1 -.->|执行| SC2
    N2 -.|跳过| SC1
    N3 -.|跳过| SC2
```

---

## 组件设计

### 1. 集群状态管理（server/clustering/cluster.go）

**职责**：管理节点注册表、心跳状态、Leader 选举。

```go
type NodeInfo struct {
    Address   string    `json:"address"`
    Alive     bool      `json:"alive"`
    LastPing  time.Time `json:"last_ping"`
    ConsecFail int      `json:"-"` // 连续失败计数（内部）
}

type ClusterState struct {
    mu        sync.RWMutex
    myAddress string
    nodes     map[string]*NodeInfo
    leader    string
    initialized bool
}
```

关键方法：
- `NewClusterState(myAddress string, knownNodes []string) *ClusterState`
- `RegisterNode(address string)` — 收到心跳时调用
- `DeregisterDeadNodes(threshold time.Duration)` — 定时清理
- `IsLeader() bool` — 比较地址字典序
- `GetHealthyNodes() []string` — 返回存活节点列表
- `PublishInfo(w http.ResponseWriter)` — `/api/cluster/info` 响应

### 2. 心跳处理器（server/controllers/cluster.go）

新增文件，实现以下端点：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/cluster/info` | GET | 返回集群状态（节点列表、leader、mode） |
| `/api/cluster/ping` | POST | 接收心跳，更新节点状态，返回当前 leader 地址 |
| `/api/cluster/leader` | GET | 返回当前 leader 节点地址（供前端/Agent 快速定位） |

`POST /api/cluster/ping` 请求体：
```json
{
  "node_id": "node-a",
  "address": "http://10.0.0.1:8080",
  "started_at": "2026-09-02T03:00:00Z"
}
```

响应：
```json
{
  "leader": "http://10.0.0.1:8080",
  "nodes": [
    {"address": "http://10.0.0.1:8080", "alive": true},
    {"address": "http://10.0.0.2:8081", "alive": true}
  ]
}
```

### 3. 客户端路由（前端 + Agent）

#### 前端（frontend/src/api/client.js）

将当前的 axios 实例改为支持多节点路由的 `ClusterClient`：

```javascript
class ClusterClient {
  constructor(urls) {
    this.nodes = urls.map(u => ({ url: u, failures: 0, disabledUntil: 0 }));
    this.currentIndex = 0;
  }

  async request(method, path, options = {}) {
    const errors = [];
    for (let i = 0; i < this.nodes.length; i++) {
      const node = this.nodes[this.currentIndex];
      this.currentIndex = (this.currentIndex + 1) % this.nodes.length;
      if (this.isNodeDisabled(node)) continue;
      try {
        const resp = await this.fetchWithTimeout(node.url, method, path, options);
        node.failures = 0;
        return resp;
      } catch (e) {
        node.failures++;
        errors.push(e.message);
        if (node.failures >= 3) {
          node.disabledUntil = Date.now() + 60_000; // 禁用 60 秒
        }
      }
    }
    throw new Error('所有后端节点不可用: ' + errors.join(', '));
  }
}
```

`vite.config.js` 中，开发服务器代理配置改为支持多节点（保留原单节点 fallback）。

环境变量 `VITE_MSMP_SERVER_URLS`：逗号分隔，如 `http://10.0.0.1:8080,http://10.0.0.2:8080`

#### Agent（agent/common/agent.go）

在 `main.go` 启动时读取 `MSMP_SERVER_URLS`（逗号分隔），构建节点列表，在上报循环中按索引轮询。失败切换逻辑与前端一致。

```go
// agent/common/agent.go 新增
type ClusterRouter struct {
    nodes      []string
    index      int
    failCount  map[int]int
}
```

### 4. 配置变更（server/config/config.go）

新增字段：

```go
type ServerConfig struct {
    Addr     string   `mapstructure:"addr"`
    Mode     string   `mapstructure:"mode"`
    Nodes    []string `mapstructure:"nodes"`  // 集群节点地址列表
    NodeID   string   `mapstructure:"node_id"` // 本节点唯一标识（可选，默认生成 UUID）
}
```

config.yaml 示例（双节点）：

```yaml
server:
  addr: ":8080"
  mode: release
  nodes:
    - "http://10.0.0.1:8080"
    - "http://10.0.0.2:8080"
  node_id: "node-a"  # 每个节点配置不同 ID

db:
  driver: postgres
  dsn: "host=10.0.0.1 user=msmp password=msmp dbname=msmp sslmode=disable"
```

### 5. main.go 改动

```go
// 新增
import "MSMP/server/clustering"

func main() {
    // ... 现有初始化 ...

    // 初始化集群状态
    clusterState := clustering.NewClusterState(cfg.Server.Addr, cfg.Server.Nodes)
    
    // 仅 leader 启动调度器
    if clusterState.IsLeader() {
        go controllers.StartOfflineChecker(cfg.Agent.OfflineAfterSec)
        controllers.StartCollectorScheduler()
        go clusterState.StartHeartbeatLoop() // 主动发心跳
    } else {
        go clusterState.JoinCluster() // follower 加入集群并监听心跳
    }

    // 注册集群路由
    mux.HandleFunc("/api/cluster/info", func(w http.ResponseWriter, r *http.Request) {
        clusterState.PublishInfo(w)
    })
    mux.HandleFunc("/api/cluster/ping", func(w http.ResponseWriter, r *http.Request) {
        controllers.ClusterPingHandler(w, r, clusterState)
    })
    mux.HandleFunc("/api/cluster/leader", func(w http.ResponseWriter, r *http.Request) {
        clusterState.PublishLeader(w)
    })

    // ... 其余保持不变 ...
}
```

---

## 数据模型

无需新增数据库表。集群状态全部内存维护，通过心跳协议同步。

节点持久化信息（仅存储在内存）：
```go
type NodeInfo struct {
    Address    string    `json:"address"`
    NodeID     string    `json:"node_id"`
    Alive      bool      `json:"alive"`
    LastPingAt time.Time `json:"last_ping_at"`
    FailCount  int       `json:"-"`
}
```

---

## 正确性属性

1. **单 Leader 不变量**：任意时刻，集群中只有一个节点认为自己是 leader。
   - 证明：基于地址字典序的全局一致比较，所有节点对同一节点列表得出相同排序。

2. **无重复采集**：CollectorScheduler 只在 leader 上运行，follower 不启动。
   - 保证：`clusterState.IsLeader()` 在 leader/follower 间返回确定性结果。

3. **故障转移安全**：当前节点失败时，客户端切换到下一个健康节点，不会丢失正在进行的 Agent 上报。
   - 保证：每次请求独立路由，无长连接依赖。

4. **幂等心跳**：重复收到同一节点的心跳不会导致状态混乱。
   - 保证：`RegisterNode` 使用 upsert 语义（存在则更新，不存在则插入）。

---

## 错误处理

| 场景 | 处理策略 |
|------|---------|
| 节点心跳超时 | 标记 `unreachable`，从活跃列表移除，其他节点不再向其路由 |
| Leader 宕机 | follower 检测超时后重新选举，选出新的 leader 启动定时任务 |
| 所有节点不可达 | Agent/前端进入指数退避重试，最多等待 60s |
| 节点重启后加入 | 发送 ping 触发注册，新节点等待 leader 指令 |
| 网络分区 | 分区内各自维持最小可用，分区恢复后自动合并 |

---

## 实施计划

### 阶段一：服务端集群基础（5 个子任务）
1. 新建 `server/clustering/cluster.go`，实现 `ClusterState` 结构体和 Leader 选举
2. 新增 `server/controllers/cluster.go`，实现 `/api/cluster/{info,ping,leader}` 三个端点
3. 修改 `server/config/config.go`，添加 `Nodes` 和 `NodeID` 字段
4. 修改 `server/main.go`，根据集群模式决定是否启动调度器
5. 更新 `config.yaml` 示例，添加双节点配置

### 阶段二：客户端路由（4 个子任务）
6. 修改 `frontend/src/api/client.js`，实现 `ClusterClient` 类（多节点轮询 + 失败切换）
7. 修改 `frontend/vite.config.js`，添加 `VITE_MSMP_SERVER_URLS` 环境变量支持
8. 修改 `agent/common/agent.go`，添加 `ClusterRouter` 结构体和故障转移逻辑
9. 修改 `agent/main.go`，读取 `MSMP_SERVER_URLS` 环境变量

### 阶段三：验证与文档（3 个子任务）
10. 编写 `deploy-multi-node.md` 部署指南（含 Docker Compose 示例）
11. 本地启动两个服务器实例验证心跳和 Leader 选举
12. 更新 `docs/ARCHITECTURE.md` 中的架构图和数据流图
