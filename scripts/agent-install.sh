#!/usr/bin/env bash
# MSMP Agent 安装脚本：安装 agent 二进制与 systemd 服务。
# 用法：解压发行包后，在包目录内以 root 执行 ./agent-install.sh
#       环境变量配置写入 /etc/msmp-agent.env（首次安装自动生成）
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/msmp-agent}"
SERVICE_NAME="msmp-agent"
ENV_FILE="${ENV_FILE:-/etc/msmp-agent.env}"

if [ "$(id -u)" -ne 0 ]; then
  echo "错误：请以 root 运行（sudo ./agent-install.sh）" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

BIN=""
for f in msmp-agent-linux-* msmp-agent.exe msmp-agent; do
  if [ -f "$SCRIPT_DIR/$f" ]; then
    BIN="$f"
    break
  fi
done
if [ -z "$BIN" ]; then
  echo "错误：未找到 msmp-agent 二进制" >&2
  exit 1
fi
echo "安装 $BIN -> $INSTALL_DIR/msmp-agent"

mkdir -p "$INSTALL_DIR"
install -m 0755 "$SCRIPT_DIR/$BIN" "$INSTALL_DIR/msmp-agent"

if [ ! -f "$ENV_FILE" ]; then
  cat >"$ENV_FILE" <<EOF
# MSMP Agent 环境变量配置
MSMP_SERVER_URLS=http://your-server:8080
AGENT_UUID=$(hostname)
AGENT_TOKEN=your-agent-token
EOF
  chmod 600 "$ENV_FILE"
  echo "已生成配置 $ENV_FILE，请编辑后重启服务：systemctl restart ${SERVICE_NAME}"
else
  echo "保留已有配置 $ENV_FILE"
fi

cat >"/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=MSMP Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=${ENV_FILE}
ExecStart=${INSTALL_DIR}/msmp-agent
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now "$SERVICE_NAME"

echo
echo "安装完成。常用命令："
echo "  systemctl status ${SERVICE_NAME}"
echo "  journalctl -u ${SERVICE_NAME} -f"
echo "  环境变量：${ENV_FILE}（修改后需 systemctl restart ${SERVICE_NAME}）"
