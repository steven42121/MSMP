# Security Policy

MSMP 是运维管理平台，直接管理生产服务器。安全报告对我们至关重要。

## 支持版本

| 版本 | 支持状态 |
|------|---------|
| v0.1.0+ | 接收安全更新 |
| 旧版本 | 不再支持，请升级到最新 |

## 报告漏洞

**请勿通过公开 Issue 报告安全漏洞。**

通过以下渠道提交：

- Email: [your-security-contact@email.com]
- 或创建 [私有 Issue](../../issues/new?template=security.md)（仅维护者可见）

我们承诺：

- 72 小时内确认收到报告
- 优先修复严重漏洞
- 修复后公布 CVE（如适用）并致谢报告者

## 安全最佳实践（部署时）

1. **修改 JWT 密钥**：`jwt.secret` 必须改为强随机值
2. **配置 credentialkey**：`security.credentialkey` 为 32 字节 AES 密钥
3. **启用 HTTPS**：反向代理层配置 TLS，不要裸跑 HTTP
4. **网络隔离**：后端端口 `:8080` 仅对内网开放
5. **定期备份数据库**：`server/msmp.db` 含所有凭据密文

## 已知安全风险

暂无。如发现新问题请及时报告。
