# models — 数据模型

本目录包含所有 GORM 模型定义，对应数据库表结构。

## 结构

```
models/
└── models.go   # 全部实体定义（Tenant、User、Host、MetricSample 等 13 个模型）
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `models.go` | 全部 GORM 模型，含表结构 tag、索引定义、软删除支持 |

## 模型清单

| 模型 | 表名 | 描述 |
|------|------|------|
| `Tenant` | `tenants` | 租户 |
| `User` | `users` | 用户（含密码哈希） |
| `Host` | `hosts` | 主机 |
| `HostTag` | `host_tags` | 主机标签 |
| `AgentToken` | `agent_tokens` | Agent 注册 Token |
| `AssetSnapshot` | `asset_snapshots` | 资产快照 |
| `MetricSample` | `metric_samples` | 指标样本 |
| `HostEvent` | `host_events` | 主机事件 |
| `AlertRule` | `alert_rules` | 告警规则 |
| `AuditLog` | `audit_logs` | 审计日志 |
| `Setting` | `settings` | 系统设置 |
| `Task` | `tasks` | 远程任务 |
| `ChannelBinding` | `channel_bindings` | 采集渠道绑定 |
| `CollectEvent` | `collect_events` | 采集事件 |

## 依赖

**本模块依赖**:
- `gorm.io/gorm` — ORM 框架
- `time` — 时间类型

**依赖本模块的**:
- `server/db/` — AutoMigrate 列表
- `server/controllers/` — 所有 handler
- `server/collectors/` — CollectEvent 写入

## 规范

### 命名约定

- 模型名 PascalCase，对应表名 snake_case（GORM 自动转换）
- 主键统一使用 `uint` + `gorm:"primaryKey"`
- 软删除统一使用 `gorm.DeletedAt` + `gorm:"index"`
- 敏感字段（如 PasswordHash、Credential）使用 `json:"-"` 排除响应

### 索引

- 租户隔离字段 `tenant_id` 加索引
- 高频查询字段（如 `host_id` + `timestamp`）加复合索引
- 唯一约束用于防重复（如 `uuid`、`token`、`slug`）
