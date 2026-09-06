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

### 工程化
- 完整 README、MIT License
- 单元测试（危险命令检测、凭据加解密、告警引擎）
- GitHub Actions CI（构建 + 测试 + 静态检查）
- GitHub Releases（tag push 自动打包多平台二进制）
- Dockerfile + docker-compose 容器化部署
- Makefile 统一构建命令

[unreleased]: https://github.com/steven42121/MSMP/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/steven42121/MSMP/releases/tag/v0.1.0