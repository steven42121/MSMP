#!/usr/bin/env bash
# MSMP Server 安装脚本：安装二进制、前端静态文件与 systemd 服务。
# 用法：解压发行包后，在包目录内以 root 执行 ./install.sh
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/msmp}"
SERVICE_NAME="msmp-server"

if [ "$(id -u)" -ne 0 ]; then
  echo "错误：请以 root 运行（sudo ./install.sh）" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *)
    echo "错误：不支持的架构 $ARCH" >&2
    exit 1
    ;;
esac

BIN=""
for f in msmp-server-linux-* msmp-server; do
  if [ -x "$f" ] || [ -f "$f" ]; then
    BIN="$f"
    break
  fi
done
if [ -z "$BIN" ]; then
  echo "错误：未找到 msmp-server 二进制" >&2
  exit 1
fi
echo "安装 $BIN -> $INSTALL_DIR/（系统架构 $ARCH）"

mkdir -p "$INSTALL_DIR"
install -m 0755 "$BIN" "$INSTALL_DIR/msmp-server"

if [ -d frontend-dist ]; then
  rm -rf "$INSTALL_DIR/frontend-dist"
  cp -r frontend-dist "$INSTALL_DIR/frontend-dist"
fi

if [ -f config.yaml ] && [ ! -f "$INSTALL_DIR/config.yaml" ]; then
  cp config.yaml "$INSTALL_DIR/config.yaml"
elif [ -f config.yaml ]; then
  echo "保留已有配置 $INSTALL_DIR/config.yaml"
fi

cat >"/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=MSMP Server
After=network.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/msmp-server
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now "$SERVICE_NAME"

echo
echo "安装完成。常用命令："
echo "  systemctl status ${SERVICE_NAME}"
echo "  journalctl -u ${SERVICE_NAME} -f"
echo "  配置文件：${INSTALL_DIR}/config.yaml"
