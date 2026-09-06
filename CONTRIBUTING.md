# Contributing to MSMP

感谢你对 MSMP 的关注！本文档说明如何参与项目贡献。

## 开发环境搭建

### 前置要求

- Go 1.26+
- Node.js 20+
- SQLite（默认，无需额外配置）

### 本地运行

```bash
# 后端（debug 模式）
cd server && go run main.go

# 前端（另一个终端）
cd frontend && npm install && npm run dev
```

前端开发服务器监听 `:5173`，已配置 `/api` 反向代理到后端 `:8080`。

### 构建

```bash
make build          # 构建 server + agent + frontend
make test           # 运行所有单元测试
make vet            # go vet 静态检查
```

### Agent 交叉编译

```bash
# Linux amd64
cd agent && GOOS=linux GOARCH=amd64 go build -o msmp-agent-linux-amd64 .

# Windows amd64
cd agent && GOOS=windows GOARCH=amd64 go build -o msmp-agent.exe .
```

## 分支规范

```
main             # 主分支，保持稳定
feature/<name>   # 新功能
fix/<name>       # bug 修复
chore/<name>     # 工程化变更
docs/<name>      # 文档更新
```

## Commit 规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <description>

# 示例
feat(monitor): add GPU and temperature display
fix(alerts): suppress duplicate alerts within window
chore: release v0.1.0
```

**type 列表**：`feat` / `fix` / `chore` / `docs` / `refactor` / `test`

## 提 PR 前检查清单

- [ ] `make test` 通过
- [ ] `make vet` 无警告
- [ ] 前端 `npm run build` 无报错
- [ ] 变更涉及 API 时同步更新 `.monkeycode/docs/INTERFACES.md`
- [ ] 新功能在 CHANGELOG.md `[Unreleased]` 下记录

## 代码风格

- Go：遵循 `gofmt` / `goimports`，无特殊 lint 规则
- 前端：使用项目已有的 ESLint 配置（`frontend/.eslintrc*`）
- 不添加不必要的注释，代码自解释优先
