# 接口文档

## 认证

所有需要认证的接口在请求头中携带：
```
Authorization: Bearer <jwt_token>
```

Token 通过 `/api/auth/login` 获取：
```json
POST /api/auth/login
{ "username": "admin", "password": "..." }

// 响应
{ "token": "eyJ...", "user": {...}, "expire_at": "2026-09-02T..." }
```

刷新 Token：
```
POST /api/auth/refresh（需有效 Authorization）
```

未认证请求返回 `401 {"error":"unauthorized"}`。

---

## Agent 接口

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/agents/register` | Agent 首次注册，返回 host UUID |
| POST | `/api/agents/heartbeat` | 心跳，携带 UUID |
| POST | `/api/agents/assets` | 资产全量上报（每 5 分钟） |
| POST | `/api/agents/metrics` | 指标上报（每 60 秒） |
| GET | `/api/agents/tasks/{token}` | 任务轮询 |

请求体示例（指标）：
```json
{
  "uuid": "xxx",
  "cpu_percent": 23.5,
  "mem_percent": 45.2,
  "mem_used": 7516192768,
  "mem_total": 16777216000,
  "disk_used": 53687091200,
  "disk_total": 214748364800,
  "net_rx_bps": 1234567890,
  "net_tx_bps": 987654321,
  "load1": 0.5,
  "load5": 0.3,
  "load15": 0.2,
  "uptime_sec": 86400
}
```

---

## 主机管理

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/hosts` | 列出当前租户所有主机 |
| POST | `/api/hosts` | 手动添加主机 |
| DELETE | `/api/hosts` | 批量删除（`{"ids":[...]}`） |
| GET | `/api/hosts/{uuid}` | 主机详情 |
| PUT | `/api/hosts/{uuid}` | 更新部分字段（hostname, os_version） |
| DELETE | `/api/hosts/{uuid}` | 删除主机 |

### 子资源

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/hosts/{uuid}/metrics?limit=N` | 监控样本（默认 60 条） |
| GET | `/api/hosts/{uuid}/events?limit=N` | 事件列表 |
| GET | `/api/hosts/{uuid}/assets?limit=N` | 资产快照 |
| GET | `/api/hosts/{uuid}/tags` | 标签列表 |
| POST | `/api/hosts/{uuid}/tags` | 添加标签 `{key, value}` |
| DELETE | `/api/hosts/{uuid}/tags/{id}` | 删除标签 |
| GET | `/api/hosts/{uuid}/channels` | 采集渠道列表（脱敏） |
| POST | `/api/hosts/{uuid}/channels` | 创建采集渠道（自动触发探测） |

---

## 采集渠道管理（无 Agent 采集）

**权限**: 全部需 `admin` 角色。

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/channels/ssh-keypair?host_uuid=xxx` | 生成 SSH 密钥对，返回公钥与一键安装命令 |
| GET | `/api/channels/{id}` | 渠道详情（脱敏） |
| PUT | `/api/channels/{id}` | 更新渠道参数（priority/enabled/address/secret） |
| DELETE | `/api/channels/{id}` | 删除渠道并清理凭据 |
| POST | `/api/channels/{id}/probe` | 手动触发连通性探测 |

### 创建渠道请求体

```json
{
  "type": "ssh",
  "address": "10.0.0.1:22",
  "auth_mode": "password",
  "username": "root",
  "secret": "mypassword",
  "priority": 100,
  "enabled": true
}
```

`type` 可选值：`ssh`、`wac`、`baota`

`auth_mode` 按 type 可选值：
- SSH：`password`、`private_key`、`generated_key`
- WAC：`gateway`
- 宝塔：`api_key`、`gateway`

### 探测响应

```json
{ "ok": true, "os": "linux", "host": "my-host" }
// 失败时
{ "ok": false, "error": "unreachable", "detail": "..." }
```

---

## 监控数据

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/metrics?host_uuid=xx&duration=1h&limit=300` | 查询监控样本 |

查询参数：
- `host_uuid`：主机 UUID（可选，不传则返回空）
- `duration`：时间范围（`1h`/`6h`/`24h`，默认 `1h`）
- `limit`：最大返回条数（默认 300）

响应为 `MetricSample` 数组，按 `timestamp` 升序。

---

## 任务管理

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/tasks` | 列出当前租户任务 |
| POST | `/api/tasks` | 创建任务 `{host_id, type, command, timeout_sec}` |
| GET | `/api/tasks/{id}` | 任务详情 |
| PUT | `/api/tasks/{id}` | 取消/更新任务 |
| DELETE | `/api/tasks/{id}` | 删除任务 |

任务类型：`shell`、`restart`、`upgrade`

---

## 告警

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/alerts` | 告警列表 |
| GET | `/api/alerts/{id}` | 告警详情 |
| PUT | `/api/alerts/{id}` | 确认/静默告警 `{acknowledged, silenced_until}` |
| GET | `/api/alert-rules` | 规则列表（admin） |
| POST | `/api/alert-rules` | 创建规则（admin） |
| PUT | `/api/alert-rules/{id}` | 更新规则（admin） |
| DELETE | `/api/alert-rules/{id}` | 删除规则（admin） |

告警规则字段：
```json
{
  "name": "CPU 过高",
  "metric": "cpu_percent",
  "operator": "gt",
  "threshold": 90.0,
  "level": "warning",
  "enabled": true
}
```

`operator` 可选：`gt`、`gte`、`lt`、`lte`
`level` 可选：`info`、`warning`、`critical`

---

## 租户与用户

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/tenants` | 租户列表（admin） |
| POST | `/api/tenants` | 创建租户（admin） |
| GET | `/api/users` | 当前租户用户列表 |
| POST | `/api/users` | 创建用户 |
| GET | `/api/users/{id}` | 用户详情 |
| PUT | `/api/users/{id}` | 更新用户 |
| DELETE | `/api/users/{id}` | 删除用户 |

---

## Agent Token

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/agent-tokens` | Token 列表（admin） |
| POST | `/api/agent-tokens` | 创建 Token |
| GET | `/api/agent-tokens/{id}` | Token 详情 |
| POST | `/api/agent-tokens/{id}/revoke` | 吊销 Token |
| DELETE | `/api/agent-tokens/{id}` | 删除 Token |

---

## 审计日志

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/audit-logs` | 日志列表（admin） |

---

## 系统设置

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/settings` | 设置列表 |
| PUT | `/api/settings` | 批量更新设置 `{key: value}` |

---

## 健康检查

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/health` | 无需认证，返回 `{"status":"running","db":"ok"}` |

---

## 错误格式

所有错误响应统一格式：
```json
{ "error": "错误描述" }
```

HTTP 状态码：
- `400` — 请求参数无效
- `401` — 未认证或 Token 过期
- `403` — 角色不足
- `404` — 资源不存在
- `409` — 冲突（如同类型渠道已启用）
- `503` — 凭据服务不可用（未配置 credential_key）
- `500` — 服务端内部错误

---

## WebSSH 终端

浏览器内 SSH 终端，通过 WebSocket 代理到目标主机的 SSH 服务。

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/hosts/{uuid}/ssh` | WebSocket 升级，建立 SSH 连接 |

客户端需通过 WebSocket URL 传递 JWT token：
```javascript
const ws = new WebSocket(`/api/hosts/${uuid}/ssh?token=${token}`);
ws.send(JSON.stringify({ width: 80, height: 24 })); // 首条消息：终端尺寸
```

终端输入通过 BinaryMessage 发送，输出通过 BinaryMessage 接收。支持终端 resize（发送 `{type: "resize", width, height}`）。

前置条件：
- 主机配置了 `type=ssh` 的采集渠道
- 凭证已加密存储（AES-256-GCM）
- 用户已认证

---

## 文件管理（SFTP）

通过 SFTP 协议管理远程主机文件。所有接口需 admin 角色。

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/hosts/{uuid}/files?path=/` | 列出目录 |
| GET | `/api/hosts/{uuid}/files/download?path=/etc/hosts` | 下载文件 |
| POST | `/api/hosts/{uuid}/files/upload` | 上传文件（multipart，最大 64MB） |
| POST | `/api/hosts/{uuid}/files/mkdir` | 创建目录 `{path}` |
| POST | `/api/hosts/{uuid}/files/rename` | 重命名 `{old_path, new_path}` |
| DELETE | `/api/hosts/{uuid}/files?path=/tmp/xxx` | 删除文件或空目录 |

文件列表响应：
```json
{
  "path": "/",
  "files": [
    {"name": "etc", "path": "/etc", "size": 0, "is_dir": true, "mode": "drwxr-xr-x", "mod_time": "2026-09-01 12:00:00"}
  ],
  "count": 1
}
```

---

## 告警工程化

### 抑制规则

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/alert-suppressions` | 列表（admin） |
| POST | `/api/alert-suppressions` | 创建（admin） |
| PUT | `/api/alert-suppressions/:id` | 更新（admin） |
| DELETE | `/api/alert-suppressions/:id` | 删除（admin） |

抑制规则字段：
```json
{
  "name": "CPU 告警静默30分钟",
  "host_id": 0,
  "metric": "cpu",
  "level": "warning",
  "window_minutes": 30,
  "enabled": true
}
```
- `host_id=0` 表示全局抑制
- 同一主机+指标+级别在窗口内只产生一条告警

### 静默规则

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/alert-silences` | 列表（admin） |
| POST | `/api/alert-silences` | 创建（admin） |
| PUT | `/api/alert-silences/:id` | 更新（admin） |
| DELETE | `/api/alert-silences/:id` | 删除（admin） |

静默规则字段：
```json
{
  "name": "维护窗口静默",
  "host_id": 0,
  "label_key": "",
  "label_value": "",
  "level": "warning",
  "start_at": "2026-09-04T10:00:00Z",
  "end_at": "2026-09-04T12:00:00Z"
}
```

### 升级规则

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/alert-escalations` | 列表（admin） |
| POST | `/api/alert-escalations` | 创建（admin） |
| PUT | `/api/alert-escalations/:id` | 更新（admin） |
| DELETE | `/api/alert-escalations/:id` | 删除（admin） |

升级规则字段：
```json
{
  "name": "Warning 升级 Critical",
  "trigger_level": "warning",
  "notify_after_min": 60,
  "retry_count": 3,
  "enabled": true
}
```
- 未确认告警超过 `notify_after_min` 分钟后自动升级为 critical
- 每轮检查最多重试 `retry_count` 次

---

## ESXi vSphere 采集

### 渠道配置

创建渠道时 `type=vsphere`：
```json
{
  "type": "vsphere",
  "address": "vcenter.example.com",
  "auth_mode": "password",
  "username": "administrator@vsphere.local",
  "secret": "username:password",
  "priority": 50
}
```

采集指标：
- ESXi 主机 CPU 使用率（平均值）
- ESXi 主机内存使用率
- VM 数量（按电源状态分类）
- Datastore 空间（总量/可用量）

---

## MCP 工具接口

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/mcp/tools` | 工具定义列表（admin） |
| POST | `/api/mcp/propose` | AI 提交工具调用，返回审批或结果 |
| GET | `/api/mcp/approvals` | 待审批列表 |
| POST | `/api/mcp/approvals/:id/approve` | 批准（admin） |
| POST | `/api/mcp/approvals/:id/reject` | 拒绝 |

**安全机制：**
- `execute_command` 等危险操作必须经过人工审批
- 审批前进行危险命令检测（`rm -rf`, `dd`, 管道注入等）
- 仅 admin 可批准工具调用
- 参数净化：`service` 只允许字母数字，`log_path` 防止路径穿越
