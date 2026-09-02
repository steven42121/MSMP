# collectors — 无 Agent 采集渠道

本目录实现无 Agent 采集能力，支持通过 SSH、Windows Admin Center（WAC）网关、宝塔面板三种远程渠道获取主机性能指标。

## 结构

```
collectors/
├── channel.go   # Channel 接口定义、Registry、MetricDataLike
├── ssh.go       # SSH 渠道：/proc 系列文件解析
├── wac.go       # Windows Admin Center 网关 API 采集
└── baota.go     # 宝塔面板 API 采集
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `channel.go` | 定义 Channel 接口（Type/Probe/Collect）、MetricDataLike 数据结构、ChannelRegistry 注册表、状态常量 |
| `ssh.go` | SSH 渠道实现：建立连接、执行只读命令、解析 /proc/stat/meminfo/net/dev/loadavg/uptime |
| `wac.go` | WAC 渠道实现：调用 WAC 网关 REST API 获取性能计数器 |
| `baota.go` | 宝塔渠道实现：调用宝塔面板系统状态 API |

## 设计要点

- **Channel 接口统一抽象**: 所有渠道实现相同的 `Probe` 和 `Collect` 方法，调度器无需感知具体类型
- **错误分类标准化**: 所有失败归为 6 种枚举状态（ok/unreachable/auth_failed/denied/unsupported/parse_error）
- **凭据由外部提供**: Channel 不直接持有凭据，由 CredentialService 解密后传入
- **语义与 Agent 一致**: SSH 采集的 net_rx/net_tx 为累计字节数，与 Agent 端 IOCounters 语义对齐

## 依赖

**本模块依赖**:
- `server/models/` — ChannelBinding 模型
- `server/services/` — CredentialService 解密
- `golang.org/x/crypto/ssh` — SSH 连接（仅 ssh.go 使用）

**依赖本模块的**:
- `server/controllers/channels.go` — CollectorScheduler 调用 Channel.Collect
- `server/controllers/hosts.go` — 渠道子资源 API

## 规范

### 新增渠道

1. 实现 `Channel` 接口（Type/Probe/Collect 三个方法）
2. 在 `server/controllers/channels.go` 的 `initCollectors()` 中调用 `channelReg.Register(&YourChannel{})`
3. 前端 `HostDetail.jsx` 创建向导的 type 选项中添加新值

### 错误处理

所有 Channel 方法返回 `(result, error)`，错误通过 `classifyErr` 归类为标准状态字符串。调度器根据状态决定是否尝试下一优先级渠道。
