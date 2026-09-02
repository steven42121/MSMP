# db — 数据库层

本目录管理数据库连接和自动迁移。

## 结构

```
db/
└── db.go   # 连接初始化、AutoMigrate
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `db.go` | `Init(cfg)` 根据配置选择驱动（sqlite/postgres），执行 AutoMigrate 创建/更新所有表 |

## 依赖

**本模块依赖**:
- `server/models/` — 所有模型（传入 AutoMigrate）
- `server/config/` — 数据库配置
- `gorm.io/gorm` — ORM
- `gorm.io/driver/sqlite` / `gorm.io/driver/postgres` — 驱动

**依赖本模块的**:
- `server/main.go` — 启动时调用 `db.Init(cfg)`
- `server/controllers/` — 通过 `db.DB` 全局变量访问

## 规范

### 新增模型

1. 在 `server/models/models.go` 添加 struct
2. 在 `server/db/db.go` 的 `AutoMigrate` 列表中加入对应指针
3. 重启服务端，GORM 自动执行迁移（SQLite 支持 ALTER TABLE，PostgreSQL 需 migration 工具）
