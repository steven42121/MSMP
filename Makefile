.PHONY: build server agent frontend test lint vet run-server run-frontend docker

# 构建后端
server:
	cd server && go build -o msmp-server .

# 构建 Agent
agent:
	cd agent && go build -o msmp-agent .

# 构建前端
frontend:
	cd frontend && npm install && npm run build

# 全量构建
build: server agent frontend

# 运行单元测试
test:
	cd server && go test ./...
	cd agent && go test ./...

# 静态检查
lint: vet

vet:
	cd server && go vet ./...
	cd agent && go vet ./...

# 启动后端（开发模式）
run-server:
	cd server && go run main.go

# 启动前端（开发模式）
run-frontend:
	cd frontend && npm run dev

# Docker 部署
docker:
	docker compose up -d --build