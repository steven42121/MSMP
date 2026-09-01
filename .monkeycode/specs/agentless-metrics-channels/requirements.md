# Requirements Document

## Introduction

当前 MSMP 平台的主机性能监控依赖部署在目标服务器上的 Agent 采集并上报指标。在部分场景下（安全策略限制、客户禁止安装软件、临时巡检、托管机器无写入权限等），无法在目标服务器上安装 Agent。本特性为平台增加「无 Agent 采集渠道」能力：当主机未安装 Agent 或 Agent 不可用时，服务端可通过多种远程渠道获取该主机的性能指标与基础资产信息，渠道包括但不限于 SSH、Windows Admin Center（WAC）REST API、宝塔面板 API，并按优先级自动降级。采集到的数据写入既有的 `MetricSample` 数据链路，前端监控图表与告警联动逻辑复用现有实现。

## Glossary

- **Agent 采集**: 由部署在目标主机上的 MSMP Agent 本地采集并主动上报指标的方式
- **无 Agent 采集渠道（Channel）**: 服务端主动发起的远程指标获取方式，包括 SSH、Windows Admin Center API、宝塔面板 API
- **采集通道绑定（Channel Binding）**: 一条主机与某个渠道及其凭据参数的关联配置，含启用状态和优先级
- **降级策略（Fallback Policy）**: 当高优先级渠道采集失败时，自动尝试下一优先级渠道的规则
- **采集调度器（Collector Scheduler）**: 服务端后台任务，按固定周期对启用了无 Agent 渠道的主机发起采集
- **Ability 探测（Probe）**: 使用绑定的凭据对主机做一次连通性与权限验证，测试渠道是否可用
- **MetricSample**: 平台既有的指标样本数据模型，本特性产出数据与之对齐
- **托管凭据（Credential）**: SSH 私钥/密码、WAC Token、宝塔 API Key 等敏感信息，存储于服务端并加密

## Requirements

### Requirement 1：渠道绑定配置管理

**User Story:** AS 平台租户管理员, I want 为主机配置一个或多个无 Agent 采集渠道及其凭据, so that 当 Agent 不可用时可远程获取主机指标。

#### Acceptance Criteria

1. WHEN 租户管理员为主机创建采集通道绑定, the 系统 SHALL 保存渠道类型、连接地址、端口、凭据引用、启用状态与优先级。
2. WHEN 渠道类型为 SSH, the 系统 SHALL 要求用户提供一个凭据，凭据形式为密码或 PEM 格式私钥，并支持指定登录用户名与端口（默认 22）。
3. WHEN 渠道类型为 Windows Admin Center, the 系统 SHALL 要求用户提供 WAC 网关地址、认证方式与访问凭据。
4. WHEN 渠道类型为宝塔面板, the 系统 SHALL 要求用户提供面板地址、API 接口密钥（API Key）。
5. WHEN 查询主机可用的采集渠道列表, the 系统 SHALL 按优先级升序返回该主机的全部通道绑定，且响应中不包含凭据明文。
6. IF 主机已存在同类型的启用状态通道绑定, THEN the 系统 SHALL 拒绝创建重复绑定并返回冲突错误。
7. WHEN 租户管理员更新或删除通道绑定, the 系统 SHALL 校验操作的绑定属于当前租户。

### Requirement 2：凭据安全存储

**User Story:** AS 平台租户管理员, I want 渠道凭据被加密存储且永不明文返回, so that 敏感信息不会因接口或数据库泄露而暴露。

#### Acceptance Criteria

1. WHEN 系统保存渠道凭据, the 系统 SHALL 使用服务端配置的对称密钥对凭据内容加密后落库。
2. WHEN 任何 API 返回通道绑定或主机信息, the 系统 SHALL 对凭据字段返回掩码值或省略。
3. IF 服务端未配置加密密钥, THEN the 系统 SHALL 拒绝创建含凭据的通道绑定并返回明确错误。
4. WHEN 用户删除通道绑定, the 系统 SHALL 同步删除关联的凭据记录。

### Requirement 3：渠道连通性探测

**User Story:** AS 平台租户管理员, I want 在保存渠道配置后测试其连通性, so that 我可以确认渠道真正可用后再启用。

#### Acceptance Criteria

1. WHEN 用户对某条通道绑定发起探测, the 系统 SHALL 在 15 秒内完成一次连通性测试并返回成功或失败原因。
2. WHEN 探测成功, the 系统 SHALL 返回目标主机的可识别信息（操作系统类型、主机名）。
3. IF 探测失败, THEN the 系统 SHALL 返回可定位的失败分类（网络不可达、认证失败、权限不足、渠道服务不可用）。
4. WHEN 探测请求超时超过 15 秒, the 系统 SHALL 中止该次探测并返回超时错误。

### Requirement 4：无 Agent 指标采集调度

**User Story:** AS 平台, I want 周期性从已启用渠道的主机采集指标, so that 无 Agent 主机的监控数据与 Agent 主机保持可比性。

#### Acceptance Criteria

1. WHILE 主机存在至少一个启用状态的通道绑定且未在 5 分钟内收到 Agent 上报, the 采集调度器 SHALL 按 60 秒周期对该主机发起无 Agent 采集。
2. WHEN 采集调度器对主机执行采集, the 系统 SHALL 按通道绑定优先级从高到低依次尝试，首个成功渠道的采集结果作为本次结果。
3. IF 某渠道本次采集失败, THEN the 系统 SHALL 记录失败原因并尝试下一优先级渠道。
4. IF 全部启用渠道均采集失败, THEN the 系统 SHALL 为该主机生成一条采集失败事件记录，并将主机状态置为 offline。
5. WHEN 无 Agent 渠道采集成功, the 系统 SHALL 将指标写入与 Agent 上报相同的 MetricSample 数据链路，并更新主机 last_heartbeat 与 online 状态。
6. WHILE 主机在 5 分钟内有 Agent 上报数据, the 采集调度器 SHALL 跳过该主机的无 Agent 采集。

### Requirement 5：SSH 渠道采集能力

**User Story:** AS 平台, I want 通过 SSH 在目标主机执行只读命令获取指标, so that Linux/Unix 主机无需安装任何软件即可被监控。

#### Acceptance Criteria

1. WHEN SSH 渠道采集 CPU 指标, the 系统 SHALL 通过读取 /proc/stat 两次采样计算 CPU 使用百分比。
2. WHEN SSH 渠道采集内存指标, the 系统 SHALL 解析 /proc/meminfo 获取内存总量与使用量。
3. WHEN SSH 渠道采集磁盘指标, the 系统 SHALL 解析 df 命令输出累加各挂载点的已用与总量。
4. WHEN SSH 渠道采集网络指标, the 系统 SHALL 读取 /proc/net/dev 的累计收发字节数，与 Agent 采集语义一致。
5. WHEN SSH 渠道采集主机负载与运行时长, the 系统 SHALL 解析 /proc/loadavg 与 /proc/uptime。
6. WHEN SSH 渠道首次采集成功, the 系统 SHALL 同步更新主机资产字段（主机名、操作系统、CPU 型号与核数、内存总量、磁盘总量）。
7. IF 目标主机操作系统为非 Linux 且 SSH 渠道命令不兼容, THEN the 系统 SHALL 将本次采集标记为渠道不支持并尝试下一渠道。

### Requirement 6：Windows Admin Center 渠道采集能力

**User Story:** AS 平台, I want 通过 Windows Admin Center 网关 API 获取 Windows 主机指标, so that 已部署 WAC 的 Windows 环境无需额外安装 Agent。

#### Acceptance Criteria

1. WHEN WAC 渠道采集指标, the 系统 SHALL 通过 WAC 网关的性能计数器相关接口获取 CPU 使用率与内存总量/已用量。
2. WHEN WAC 渠道采集成功, the 系统 SHALL 将获取到的指标映射到 MetricSample 对应字段，Get 不到的字段按零值处理并标记。
3. WHEN 目标 Windows 主机已被 WAC 纳管但指标接口返回权限错误, the 系统 SHALL 将失败原因归类为权限不足。
4. IF WAC 渠道不支持获取某类指标（如网络累计字节）, THEN the 系统 SHALL 对该字段按零值写入并在采集结果元数据中标记缺项。

### Requirement 7：宝塔面板渠道采集能力

**User Story:** AS 平台, I want 通过宝塔面板 API 获取主机负载信息, so that 已安装宝塔面板的 Linux 主机可直接复用面板数据。

#### Acceptance Criteria

1. WHEN 宝塔渠道采集指标, the 系统 SHALL 调用宝塔面板系统状态类接口获取 CPU 使用率、内存使用、负载信息。
2. WHEN 宝塔面板接口返回数据结构变更导致解析失败, the 系统 SHALL 将本次采集标记为渠道解析错误并尝试下一渠道。
3. WHEN 宝塔渠道采集成功但缺少磁盘或网络指标, the 系统 SHALL 将缺失字段按零值写入并在采集结果元数据中标记缺项。

### Requirement 8：采集结果与告警联动

**User Story:** AS 平台, I want 无 Agent 渠道采集的指标走与 Agent 相同的存储与告警链路, so that 告警规则与监控图表对所有主机行为一致。

#### Acceptance Criteria

1. WHEN 无 Agent 渠道采集的指标写入 MetricSample, the 系统 SHALL 触发 evaluateMetricAlerts 告警评定逻辑。
2. WHEN 前端查询某主机监控数据, the 系统 SHALL 合并返回 Agent 与无 Agent 渠道写入的样本，前端无需感知来源差异。
3. WHEN 前端展示无 Agent 主机的监控数据, the 系统 SHALL 在主机详情中展示当前生效的采集渠道与最近一次采集时间。

### Requirement 9：渠道自动化接入方式

**User Story:** AS 平台租户管理员, I want 每种渠道提供多种自动化接入方式供我选择, so that 我可以根据目标主机的管理现状选择成本最低的接入方式。

#### Acceptance Criteria

1. WHEN 用户为 SSH 渠道创建通道绑定, the 系统 SHALL 提供三种接入方式：密码直连、用户自带私钥、平台生成密钥对并输出一键安装公钥命令。
2. WHEN 用户选择"平台生成密钥对"方式, the 系统 SHALL 生成一对 SSH 密钥，仅保存私钥（加密存储），并向用户展示可直接在目标主机执行的公钥安装命令。
3. WHEN 用户为宝塔面板渠道创建通道绑定, the 系统 SHALL 提供两种接入方式：用户粘贴既有 API Key、平台引导生成 API Key 的指引与跳转链接。
4. WHEN 用户为 Windows Admin Center 渠道创建通道绑定, the 系统 SHALL 提供网关凭据直连方式，并展示目标主机加入 WAC 纳管的引导说明。
5. WHEN 同一主机存在多种可用接入方式, the 系统 SHALL 在主机详情页按推荐顺序展示全部可选接入方式及其当前状态。
6. WHEN 通道绑定创建成功, the 系统 SHALL 自动触发一次连通性探测，并将探测结果返回给用户。

### Requirement 10：渠道审计

**User Story:** AS 平台租户管理员, I want 无 Agent 采集的关键操作被记录, so that 渠道使用行为可追溯。

#### Acceptance Criteria

1. WHEN 创建、更新、删除通道绑定或执行探测, the 系统 SHALL 写入一条审计日志，包含操作者、操作类型、渠道类型与目标主机标识。
2. IF 无 Agent 渠道连续 5 次采集失败, THEN the 系统 SHALL 自动禁用该通道绑定并生成一条告警事件。
