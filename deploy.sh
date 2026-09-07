#!/bin/bash
# MSMP 服务管理脚本 - 一键部署和守护
# 用法: ./deploy.sh [install|start|stop|restart|logs|status|watchdog]

set -u

INSTALL_DIR="/opt/msmp"
SERVICE_NAME="msmp-server"
USER="$(whoami)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

cmd="${1:-help}"

case "$cmd" in
  install)
    echo "=== 安装 MSMP 到 $INSTALL_DIR ==="
    sudo mkdir -p "$INSTALL_DIR/data" "$INSTALL_DIR/logs"
    cd "$SCRIPT_DIR"
    make build
    sudo cp dist/msmp-server "$INSTALL_DIR/"
    sudo cp server/config.yaml "$INSTALL_DIR/"
    sudo chmod 755 "$INSTALL_DIR/msmp-server"
    sudo tee "/etc/systemd/system/$SERVICE_NAME.service" > /dev/null <<'UNIT'
[Unit]
Description=MSMP Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory={{INSTALL_DIR}}
ExecStart={{INSTALL_DIR}}/msmp-server
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
UNIT
    sudo sed -i "s|{{INSTALL_DIR}}|$INSTALL_DIR|g" "/etc/systemd/system/$SERVICE_NAME.service"
    sudo systemctl daemon-reload
    sudo systemctl enable "$SERVICE_NAME"
    sudo systemctl start "$SERVICE_NAME"
    echo "✓ 安装完成，服务已启动"
    echo "  状态: sudo systemctl status $SERVICE_NAME"
    echo "  日志: sudo journalctl -u $SERVICE_NAME -f"
    ;;

  start)
    sudo systemctl start "$SERVICE_NAME" && echo "✓ 已启动"
    ;;

  stop)
    sudo systemctl stop "$SERVICE_NAME" && echo "✓ 已停止"
    ;;

  restart)
    sudo systemctl restart "$SERVICE_NAME" && echo "✓ 已重启"
    ;;

  status)
    sudo systemctl status "$SERVICE_NAME" --no-pager
    ;;

  logs)
    sudo journalctl -u "$SERVICE_NAME" -f --no-pager
    ;;

  watchdog)
    echo "=== 启动 Watchdog（后台运行）==="
    nohup "$SCRIPT_DIR/contrib/watchdog.sh" \
      "$INSTALL_DIR/msmp.pid" \
      "http://localhost:8080/api/health" \
      "30" \
      "$INSTALL_DIR/logs/watchdog.log" \
      "10" "300" \
      >> "$INSTALL_DIR/logs/watchdog.log" 2>&1 &
    echo "✓ Watchdog 已启动 (PID $!)"
    echo "  日志: tail -f $INSTALL_DIR/logs/watchdog.log"
    ;;

  uninstall)
    sudo systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    sudo systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    sudo rm -f "/etc/systemd/system/$SERVICE_NAME.service"
    sudo systemctl daemon-reload
    rm -rf "$INSTALL_DIR"
    echo "✓ 已卸载"
    ;;

  help|*)
    cat <<EOF
MSMP 服务管理

  install   - 构建并安装到 $INSTALL_DIR，注册 systemd 服务
  start     - 启动服务
  stop      - 停止服务
  restart   - 重启服务
  status    - 查看服务状态
  logs      - 实时查看日志
  watchdog  - 启动崩溃自动重启守护进程
  uninstall - 卸载服务和文件

开发模式：
  make run-server   - 运行后端
  make run-frontend - 运行前端
  make docker       - Docker 部署
EOF
    ;;
esac
