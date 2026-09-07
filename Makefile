.PHONY: build server agent frontend test lint vet run-server run-frontend \
        docker docker-up docker-down docker-logs \
        service-install service-remove service-status service-start service-stop service-restart \
        deploy clean help

# ── 构建 ────────────────────────────────────────────────────────────────────
server:
	cd server && go build -o ../dist/msmp-server .

agent:
	cd agent && GOOS=linux GOARCH=amd64 go build -o ../dist/msmp-agent-linux-amd64 .
	cd agent && GOOS=linux GOARCH=arm64 go build -o ../dist/msmp-agent-linux-arm64 .
	cd agent && GOOS=windows GOARCH=amd64 go build -o ../dist/msmp-agent-windows-amd64.exe .

frontend:
	cd frontend && npm install && npm run build

build: server agent frontend
	@echo "Build complete: dist/msmp-server, dist/msmp-agent-*"

# ── 测试 ────────────────────────────────────────────────────────────────────
test:
	cd server && go test ./...
	cd agent && go test ./...

lint: vet

vet:
	cd server && go vet ./...
	cd agent && go vet ./...

# ── 开发运行 ────────────────────────────────────────────────────────────────
run-server:
	cd server && go run main.go

run-frontend:
	cd frontend && npm run dev

run-all:
	@echo "Starting server in background..."
	cd server && go run main.go &
	@echo "Starting frontend..."
	cd frontend && npm run dev

# ── Docker ──────────────────────────────────────────────────────────────────
docker:
	docker compose up -d --build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# ── 服务部署（systemd） ─────────────────────────────────────────────────────
INSTALL_DIR ?= /opt/msmp
SERVICE_NAME ?= msmp-server

service-install: build
	@echo "Installing to $(INSTALL_DIR)..."
	sudo mkdir -p $(INSTALL_DIR)/data $(INSTALL_DIR)/logs
	sudo cp dist/msmp-server $(INSTALL_DIR)/
	sudo cp server/config.yaml $(INSTALL_DIR)/
	sudo chown -R root:root $(INSTALL_DIR)
	sudo chmod 755 $(INSTALL_DIR)/msmp-server
	sudo tee /etc/systemd/system/$(SERVICE_NAME).service > /dev/null <<EOF
[Unit]
Description=MSMP Server - Mix System Manage Platform
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$(INSTALL_DIR)
ExecStart=$(INSTALL_DIR)/msmp-server
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
	sudo systemctl daemon-reload
	@echo "Install complete. Run 'make service-start' to start."

service-remove:
	sudo systemctl stop $(SERVICE_NAME) || true
	sudo systemctl disable $(SERVICE_NAME) || true
	sudo rm -f /etc/systemd/system/$(SERVICE_NAME).service
	sudo systemctl daemon-reload
	@echo "Service removed."

service-status:
	systemctl status $(SERVICE_NAME) --no-pager

service-start:
	sudo systemctl start $(SERVICE_NAME)

service-stop:
	sudo systemctl stop $(SERVICE_NAME)

service-restart:
	sudo systemctl restart $(SERVICE_NAME)

service-logs:
	journalctl -u $(SERVICE_NAME) -f --no-pager

# ── 辅助命令 ────────────────────────────────────────────────────────────────
clean:
	rm -rf dist/
	cd server && rm -f msmp-server
	cd agent && rm -f msmp-agent*

help:
	@echo "MSMP Build Commands:"
	@echo "  make build          - Build server + agent + frontend"
	@echo "  make server         - Build server binary"
	@echo "  make agent          - Build agent (linux/amd64, linux/arm64, windows/amd64)"
	@echo "  make frontend       - Build frontend (npm run build)"
	@echo "  make test           - Run all tests"
	@echo "  make vet            - Run go vet"
	@echo "  make run-server     - Run server in dev mode"
	@echo "  make run-frontend   - Run frontend in dev mode"
	@echo "  make docker         - Build and start with docker compose"
	@echo "  make docker-logs    - Follow docker compose logs"
	@echo "  make service-install - Install as systemd service"
	@echo "  make service-status  - Check service status"
	@echo "  make service-restart - Restart service"
	@echo "  make service-logs    - Tail service logs"
	@echo "  make clean          - Remove build artifacts"
