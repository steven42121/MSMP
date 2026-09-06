# Changelog

本项目所有重要变更均记录于此。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [v0.1.0] - 2026-09-06

### 新增
- 主机监控（Agent + 无 Agent 多渠道：SSH / Windows Admin Center / 宝塔面板 / Prometheus / SNMP / WinRM / vSphere）
- 阈值告警规则 + 告警工程化（抑制 / 静默 / 升级）
- 浏览器内 WebSSH 远程终端
- SFTP 远程文件管理
- ESXi/vCenter vSphere API 采集（虚拟机列表、电源操作、数据存储）
- Proxmox VE 支持（QEMU 虚拟机 + LXC 容器列表、电源操作、节点指标聚合、数据存储）
- MCP AI 工具体系（危险操作人工审批）
- 多租户隔离、角色权限、AES-256-GCM 凭据加密、审计日志

### 安全加固
- SFTP 路径穿越修复（upload/download/mkdir/rename/delete 全路径校验）
- WebSSH Origin 校验 + admin 角色强制检查 + 并发写锁
- Agent 端点认证收紧（heartbeat/assets/metrics 需 AgentToken）
- 登录失败锁定（可配置次数/时长）+ 速率限制
- IP 白名单支持 CIDR 写法
- JWT claims 类型安全检查
- 告警抑制记录独立 type（alert_suppressed），防止自锁续期
- 告警升级精确 ID 匹配 + 升级通知 webhook
- PVE LoadAvg 类型修复（string→float64 解析）+ 空数组防护
- PVE ctx 传递修复 + InsecureSkipVerify 可配置
- PVE Storage Node JSON tag 修正
- 完整 README、MIT License
- 单元测试（危险命令检测、凭据加解密、告警引擎）
- GitHub Actions CI（构建 + 测试 + 静态检查）
- GitHub Releases（tag push 自动打包多平台二进制）
- Dockerfile + docker-compose 容器化部署
- Makefile 统一构建命令
- Prometheus 格式 `/metrics` 自监控端点（请求数/错误数/GC/主机/Agent 计数）
- 时序数据降采样与保留策略（默认 7 天降采样至 5 分钟粒度，90 天自动清理）
- 登录失败锁定（5 次失败锁定 10 分钟）+ IP 白名单（config.yaml 配置）

[unreleased]: https://github.com/steven42121/MSMP/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/steven42121/MSMP/releases/tag/v0.1.0