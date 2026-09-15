# MSMP Agent（Proxmox VE）说明

Proxmox VE 基于 Debian，本包内为静态编译的 amd64 二进制，可直接安装：

```bash
chmod +x msmp-agent-pve-amd64
sudo mkdir -p /opt/msmp-agent
sudo cp msmp-agent-pve-amd64 /opt/msmp-agent/msmp-agent
```

配置环境变量 `/etc/msmp-agent.env`：

```bash
MSMP_SERVER_URLS=http://your-server:8080
AGENT_UUID=pve-01
AGENT_TOKEN=your-agent-token
```

创建 systemd 服务 `/etc/systemd/system/msmp-agent.service`：

```ini
[Unit]
Description=MSMP Agent
After=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/msmp-agent.env
ExecStart=/opt/msmp-agent/msmp-agent
Restart=always

[Install]
WantedBy=multi-user.target
```

启动：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now msmp-agent
```

> 提示：PVE 虚拟化指标（虚拟机、存储、快照等）建议同时添加 Proxmox VE 采集渠道（面板 → 主机详情 → 添加渠道 → Proxmox VE），Agent 负责主机自身的 CPU/内存/磁盘/网络指标。