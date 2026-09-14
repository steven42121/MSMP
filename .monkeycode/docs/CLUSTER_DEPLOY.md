# 多节点集群部署

MSMP 统一使用 **PostgreSQL** 作为数据库，支持单机与多节点集群两种部署方式。多节点集群下所有 server 节点共享同一 PostgreSQL，Agent 轮询上报的数据不再分裂；节点间自动选举 leader，定时采集/降采样任务仅由 leader 执行。

## 一、一键部署（Docker Compose）

### 单机

```bash
docker compose up -d
```

### 多节点集群（3 个 server 节点）

```bash
docker compose -f docker-compose.cluster.yml up -d
```

一条命令拉起：PostgreSQL（共享库）+ 3 个 server 节点 + 前端（nginx 负载均衡）+ Agent。

### 查看状态

```bash
docker compose -f docker-compose.cluster.yml ps
docker compose -f docker-compose.cluster.yml logs -f server-node-1
```

## 二、配置项

复制 `.env.example` 为 `.env`，按需修改：

| 变量 | 说明 |
|------|------|
| `POSTGRES_PASSWORD` | PostgreSQL 密码 |
| `MSMP_JWT_SECRET` | JWT 签名密钥（所有节点必须一致） |
| `MSMP_CREDENTIAL_KEY` | 凭据加密密钥（base64 32 字节，所有节点必须一致） |

**重要**：所有 server 节点必须共享相同的 `MSMP_JWT_SECRET` 与 `MSMP_CREDENTIAL_KEY`，否则跨节点登录失效、采集渠道凭据无法解密。

## 三、手动多节点部署（非 Docker）

在每台服务器上运行 server，通过环境变量或 `config.yaml` 配置：

```bash
# 节点 1
MSMP_DB_DRIVER=postgres \
MSMP_DB_DSN="host=<pg-host> user=msmp password=xxx dbname=msmp port=5432 sslmode=disable" \
MSMP_SERVER_NODES="http://node1:8080,http://node2:8080,http://node3:8080" \
MSMP_SERVER_NODE_ID=node1 \
MSMP_SERVER_ADVERTISE_ADDR=http://node1:8080 \
MSMP_JWT_SECRET=<shared-secret> \
MSMP_SECURITY_CREDENTIALKEY=<shared-key> \
./msmp-server
```

关键环境变量（全部支持 `MSMP_` 前缀 + 下划线命名）：

| 变量 | 说明 |
|------|------|
| `MSMP_DB_DSN` | PostgreSQL DSN（所有节点指向同一库） |
| `MSMP_SERVER_NODES` | 集群全部节点地址，逗号分隔 |
| `MSMP_SERVER_NODE_ID` | 当前节点唯一标识 |
| `MSMP_SERVER_ADVERTISE_ADDR` | 当前节点在集群内的通告地址（节点间心跳/leader 选举用） |
| `MSMP_JWT_SECRET` | JWT 密钥（共享） |
| `MSMP_SECURITY_CREDENTIALKEY` | 凭据加密密钥（共享） |

## 四、多节点语义

- **数据库**：统一 PostgreSQL，`db.driver` 默认即为 `postgres`（SQLite 仅保留用于本地快速验证，生产部署统一 PG）。
- **Leader 选举**：节点按通告地址字典序最小者为 leader，leader 执行采集调度器、时序降采样与清理。
- **负载均衡**：前端 nginx 对 `/api` 与 `/ws` 按轮询分发到各节点，并自动剔除故障节点。
- **故障转移**：Agent 通过 `MSMP_SERVER_URLS` 轮询上报，单节点故障自动切换到其他节点（连续失败 3 次熔断 60 秒）。

## 五、从 SQLite 迁移

历史版本使用 SQLite 时，可迁移到 PostgreSQL。迁移脚本位于 `server/migrate_sqlite_to_pg.py` 与 `server/migrate_bool_fix.py`：

```bash
cd server
pip install psycopg2-binary
python3 migrate_sqlite_to_pg.py
python3 migrate_bool_fix.py
```