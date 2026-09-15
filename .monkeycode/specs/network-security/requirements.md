# Requirements Document

## Introduction

MSMP 平台已采集主机端口清单（`host_ports`）与公网 IP（`hosts.public_ip`），并具备 SSH 采集渠道与 Agent 命令下发能力。本特性为平台新增「网络安全」模块，提供端口暴露风险审计、主机安全基线检查、防火墙管理三项能力，帮助运维人员识别并收敛主机网络暴露面。

## Glossary

- **端口暴露风险（Port Risk）**: 危险服务（SSH/数据库/缓存等）监听对外地址或主机具备公网 IP 时形成的暴露风险。
- **安全基线检查（Baseline Check）**: 对主机 SSH 配置、防火墙状态、SELinux、空密码账户等安全项执行检查并给出评分与加固建议。
- **防火墙管理（Firewall Mgmt）**: 查看主机防火墙规则、开关端口（基于 firewalld / ufw）。
- **高风险端口表（Risk Port Table）**: 内置的危险端口 → 服务名 → 风险等级映射。

## Requirements

### Requirement 1：端口暴露风险审计

**User Story:** AS 平台运维人员, I want 查看所有主机的危险端口暴露情况, so that 我可以及时收敛暴露面。

#### Acceptance Criteria

1. WHEN 运维人员请求风险报告, the 系统 SHALL 汇总所有在线主机的端口清单（host_ports），识别命中内置高风险端口表的条目。
2. WHEN 端口监听地址为对外地址（0.0.0.0 / ::）或主机具有公网 IP, the 系统 SHALL 将风险等级提升为 critical。
3. WHEN 命中高风险端口表, the 系统 SHALL 返回该端口对应的服务名、风险等级与加固建议。
4. WHEN 无任何风险, the 系统 SHALL 返回空列表。

### Requirement 2：安全基线检查

**User Story:** AS 平台运维人员, I want 对某台主机执行安全基线检查, so that 我可以了解主机的安全配置状态与不足。

#### Acceptance Criteria

1. WHEN 运维人员对主机发起基线检查, the 系统 SHALL 通过该主机的 SSH 采集渠道执行检查命令。
2. WHEN 主机未配置 SSH 渠道或连接失败, the 系统 SHALL 返回明确错误（而非崩溃）。
3. WHEN 检查完成, the 系统 SHALL 返回分条目的检查结果（项、状态、详情、建议）。
4. WHEN 检查项不适用（如非 RHEL 系统无 SELinux）, the 系统 SHALL 标记为 N/A。

### Requirement 3：防火墙管理

**User Story:** AS 平台运维人员, I want 查看并调整主机的防火墙规则, so that 我可以放行或封禁端口。

#### Acceptance Criteria

1. WHEN 运维人员查询主机防火墙, the 系统 SHALL 识别防火墙类型（firewalld / ufw）并返回当前规则列表。
2. WHEN 运维人员请求开启某端口, the 系统 SHALL 通过 SSH 执行对应防火墙命令并返回结果。
3. IF 防火墙类型无法识别, THEN the 系统 SHALL 返回不支持的错误信息。

## Out of Scope

- 主动端口扫描、漏洞扫描、渗透测试（平台禁止）。
- 主机入侵检测（HIDS）。
- 规则自动下发与定时巡检（后续迭代）。