# services — 共享服务

本目录存放跨控制器复用的业务服务，目前仅包含凭据加解密服务。

## 结构

```
services/
└── credential.go   # AES-256-GCM 凭据加密服务 + SSH 密钥对生成
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `credential.go` | CredentialService：AES-256-GCM 加密/解密；GenerateSSHKeypair：生成 OpenSSH 格式密钥对 |

## 设计要点

- **密钥从配置加载**: `NewCredentialService(cfg)` 从 `config.Security.CredentialKey` 读取 base64 编码的 32 字节密钥
- **密钥缺失友好报错**: 未配置时返回 `ErrCredentialKeyMissing`，API 返回 503
- **SSH 密钥对生成**: `GenerateSSHKeypair()` 返回 OpenSSH 公钥和 PEM 私钥，私钥由 CredentialService 加密存储
- **纯函数式**: 服务无状态（除密钥外），线程安全

## 依赖

**本模块依赖**:
- `server/config/` — 读取配置
- `golang.org/x/crypto/ssh` — SSH 密钥对生成

**依赖本模块的**:
- `server/controllers/channels.go` — 凭据加密/解密
- `server/collectors/` — 通过 CredentialProvider 接口间接使用

## 规范

### 错误处理

所有公开方法遇到密钥缺失或解密失败时返回 error，调用方应检查并返回合适的 HTTP 状态码（通常为 503）。

### 安全性

- 绝不输出明文密钥或解密后的凭据到日志
- 生成密钥对时 RSA 2048 位，满足基本安全要求
