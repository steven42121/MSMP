# MSMP Agent（ESXi）说明

ESXi 是 VMware 虚拟化平台，**推荐使用 vSphere 采集渠道**（面板 → 主机详情 → 添加渠道 → vSphere），无需在 ESXi 上安装 Agent，可采集虚拟机、存储、网络、快照等完整虚拟化数据。

如需在 ESXi 上直接运行 Agent（可选），本包内为静态编译的 x86_64 二进制，可在 ESXi 的 busybox shell 中运行：

```sh
chmod +x msmp-agent-esxi-amd64
MSMP_SERVER_URLS=http://your-server:8080 AGENT_UUID=esxi-01 AGENT_TOKEN=xxx ./msmp-agent-esxi-amd64
```

自启动可写入 `/etc/rc.local.d/local.sh`。

> 注意：ESXi 的 `/proc` 伪文件系统不完整，Agent 采集的部分指标（CPU/内存/磁盘明细）在 ESXi 上可能不可用；虚拟化指标请使用 vSphere 渠道获取。