# Changelog

本项目所有重要变更均记录于此。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增
- 主机监控（Agent + 无 Agent 多渠道：SSH / Windows Admin Center / 宝塔面板 / Prometheus / SNMP / WinRM / vSphere）
- 阈值告警规则 + 告警工程化（抑制 / 静默 / 升级）
- 浏览器内 WebSSH 远程终端
- SFTP 远程文件管理
- ESXi/vCenter vSphere API 采集
- MCP AI 工具体系（危险操作人工审批）
- 多租户隔离、角色权限、AES-256-GCM 凭据加密、审计日志

### 工程化
- 完整 README、MIT License
- 单元测试（危险命令检测、凭据加解密、告警引擎）
- GitHub Actions CI（构建 + 测试 + 静态检查）
- Dockerfile + docker-compose 容器化部署
- Makefile 统一构建命令