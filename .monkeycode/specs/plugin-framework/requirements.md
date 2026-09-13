# Requirements Document

## Introduction

MSMP 平台当前的可扩展点分散且不统一：采集渠道（`collectors.Channel`）、可用性探测（HTTP/TCP/SSL）、以及告警通知（单一 Webhook）各自为政，缺少一个统一的「插件」入口，用户无法在一个界面查看、配置、启停所有可扩展能力。

本特性为平台引入**通用插件框架**：将通知渠道、采集渠道、可用性探测统一纳管，提供统一的插件注册中心、插件实例配置、启停与测试能力，并落地第一批**通知渠道插件**（Webhook / 钉钉 / 飞书 / 企业微信 / 邮件 SMTP / Slack）。插件采用**内置注册编译**方式（接口化 + 注册表 + 编译进主程序），保证跨平台（linux amd64/arm64、windows amd64）稳定分发，后续可平滑升级为动态加载。

## Glossary

- **插件（Plugin）**: MSMP 平台的可扩展能力单元，分为通知（notifier）、采集（collector）、探测（probe）三类，均有唯一标识、展示名、描述与配置项元数据。
- **插件实例（Plugin Instance）**: 用户配置并启用的某个插件的具体实例，含该插件的配置参数，租户隔离。
- **通知插件（Notifier Plugin）**: 负责将告警/事件分发到第三方渠道（Webhook、钉钉、飞书、企业微信、邮件、Slack）的插件类型。
- **配置项元数据（Config Schema）**: 描述一个插件实例需要填写的配置字段（字段名、标签、类型、是否必填、是否为敏感项），供前端动态渲染配置表单。
- **插件注册中心（Plugin Registry）**: 服务端维护的、按类型分组的内置插件注册表。
- **告警分发（Notify）**: 告警引擎在产生告警/升级/离线事件时，将通知消息投递给全部启用的通知插件实例。

## Requirements

### Requirement 1：插件元数据查询

**User Story:** AS 平台租户管理员, I want 查询平台可用的全部插件及其配置项说明, so that 我可以了解有哪些可扩展能力并据此配置插件实例。

#### Acceptance Criteria

1. WHEN 租户管理员请求插件列表, the 系统 SHALL 返回全部内置插件，每个插件含唯一标识、类型（notifier/collector/probe）、展示名、描述与配置项元数据。
2. WHEN 返回的配置项为敏感项（如密钥、令牌）, the 系统 SHALL 在元数据中标记该字段为敏感项。
3. WHEN 返回的配置项存有默认值, the 系统 SHALL 在元数据中携带默认值。

### Requirement 2：通知插件实例管理

**User Story:** AS 平台租户管理员, I want 创建、查看、更新、启用与删除通知插件实例, so that 我可以为团队配置多个告警分发渠道。

#### Acceptance Criteria

1. WHEN 租户管理员创建通知插件实例, the 系统 SHALL 校验插件标识与类型合法、配置字段满足该插件的必填约束，并保存实例（租户隔离）。
2. WHEN 实例配置含敏感字段, the 系统 SHALL 使用服务端凭据密钥加密后落库，且查询响应中不返回明文。
3. WHEN 租户管理员启停或更新实例, the 系统 SHALL 校验该实例属于当前租户。
4. WHEN 租户管理员删除实例, the 系统 SHALL 软删除该实例并从告警分发中排除。

### Requirement 3：通知插件测试

**User Story:** AS 平台租户管理员, I want 对某个通知插件实例发起一次测试通知, so that 我可以确认渠道连通性与凭据正确。

#### Acceptance Criteria

1. WHEN 租户管理员触发实例测试, the 系统 SHALL 向该实例对应渠道发送一条测试消息，并返回成功或失败原因。
2. IF 渠道返回错误（如鉴权失败、网络不可达）, THEN the 系统 SHALL 将失败原因返回给调用方且不写入告警事件。

### Requirement 4：告警分发到全部启用的通知插件

**User Story:** AS 平台运维人员, I want 平台在产生告警时自动分发到全部启用的通知渠道, so that 重要事件能及时触达。

#### Acceptance Criteria

1. WHEN 告警引擎创建告警、升级事件或离线事件, the 系统 SHALL 将通知消息投递给该租户全部启用的通知插件实例。
2. IF 某个通知插件投递失败, THEN the 系统 SHALL 记录日志并继续分发到其余插件，单个渠道失败不阻断整体流程。
3. WHEN 租户未配置任何启用的通知实例, the 系统 SHALL 保持现有默认 Webhook 行为（若配置了 `notification.webhookurl`）以保证向后兼容。

### Requirement 5：采集与探测纳管

**User Story:** AS 平台租户管理员, I want 在插件列表中看到采集渠道与探测类型的既有能力, so that 插件框架提供统一视图。

#### Acceptance Criteria

1. WHEN 请求插件列表, the 系统 SHALL 在结果中包含既有的采集渠道（SSH/WAC/宝塔/Prometheus/SNMP/WinRM/vSphere/PVE/1Panel）与探测类型（HTTP/TCP/SSL）元数据。
2. WHEN 已存在的采集渠道与探测流程, the 系统 SHALL 保持现有行为不变（复用既有实现，仅暴露元数据）。

## Out of Scope

- 动态加载（Go plugin / 外部进程插件）——本期采用内置注册编译，动态加载列为后续演进方向。
- 告警规则的「按规则选择通知渠道」——本期为租户级全部启用实例分发，规则级渠道选择后续迭代。
- 插件市场 / 远程插件仓库。