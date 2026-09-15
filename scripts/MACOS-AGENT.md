# MSMP Agent（macOS）安装说明

## 1. 配置环境变量

```bash
export MSMP_SERVER_URLS=http://your-server:8080
export AGENT_UUID=$(hostname)
export AGENT_TOKEN=your-agent-token
```

## 2. 运行方式

### 方式 A：前台运行（快速验证）

```bash
chmod +x msmp-agent-darwin-amd64
./msmp-agent-darwin-amd64
```

Apple Silicon（M 系列）使用 `msmp-agent-darwin-arm64`。

### 方式 B：launchd 服务（推荐）

创建 `~/Library/LaunchAgents/com.msmp.agent.plist`：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.msmp.agent</string>
  <key>ProgramArguments</key>
  <array>
    <string>/opt/msmp/msmp-agent-darwin-amd64</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>MSMP_SERVER_URLS</key><string>http://your-server:8080</string>
    <key>AGENT_UUID</key><string>your-mac</string>
    <key>AGENT_TOKEN</key><string>your-agent-token</string>
  </dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
</dict>
</plist>
```

加载并启动：

```bash
launchctl load ~/Library/LaunchAgents/com.msmp.agent.plist
launchctl start com.msmp.agent
```

卸载：

```bash
launchctl unload ~/Library/LaunchAgents/com.msmp.agent.plist
```