# MSMP Agent（Windows）安装说明

## 1. 配置环境变量

PowerShell（当前用户）：

```powershell
[Environment]::SetEnvironmentVariable("MSMP_SERVER_URLS", "http://your-server:8080", "Machine")
[Environment]::SetEnvironmentVariable("AGENT_UUID", "$env:COMPUTERNAME", "Machine")
[Environment]::SetEnvironmentVariable("AGENT_TOKEN", "your-agent-token", "Machine")
```

## 2. 运行方式

### 方式 A：前台运行（快速验证）

```powershell
.\msmp-agent-windows-amd64.exe
```

### 方式 B：注册为 Windows 服务（推荐）

以管理员身份打开 PowerShell：

```powershell
sc.exe create MSMP-Agent binPath= "C:\msmp-agent\msmp-agent-windows-amd64.exe" start= auto
sc.exe description MSMP-Agent "MSMP Monitoring Agent"
sc.exe failure MSMP-Agent reset= 86400 actions= restart/10000/restart/10000/restart/60000
sc.exe start MSMP-Agent
```

查看状态与日志：

```powershell
sc.exe query MSMP-Agent
Get-EventLog -LogName Application -Source MSMP-Agent -Newest 20
```

卸载服务：

```powershell
sc.exe stop MSMP-Agent
sc.exe delete MSMP-Agent
```
