# agent — 主机采集 Agent

本目录是部署在目标主机上的独立采集进程，负责本地指标采集和上报。

## 结构

```
agent/
├── main.go             # 入口：解析参数、注册、启动 MainLoop
├── common/
│   ├── agent.go        # MetricData、HeartbeatData、RegisterData 定义，上报 HTTP 函数
│   ├── collector.go    # CollectMetrics 函数指针（平台分发）
│   ├── platform_posix.go     # POSIX 平台分发 CollectMetrics → posix.CollectMetrics
│   └── platform_windows.go   # Windows 平台分发 CollectMetrics → win.CollectMetrics
├── posix/
│   └── collect.go      # Linux/macOS 采集：gopsutil cpu/mem/disk/net/load/host
└── win/
    └── collect.go      # Windows 采集：WMI/PerformanceCounter
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `main.go` | 命令行参数解析、注册/心跳/资产/指标上报循环、信号处理 |
| `common/agent.go` | 数据结构定义（MetricData 等）、HTTP 上报函数、MainLoop 调度逻辑 |
| `posix/collect.go` | gopsutil v3 采集：CPU 使用率、内存、磁盘、网络、负载、运行时长 |
| `win/collect.go` | Windows 专用采集：Win32_Processor、Win32_OperatingSystem 等 WMI 查询 |

## 设计要点

- **MainLoop 调度**: 心跳 30s、指标上报 60s、资产全量 5min、任务轮询 10s，通过时间差计算控制节奏
- **平台抽象**: `common/collector.go` 的 `CollectMetrics` 函数指针由 `platform_*.go` 在编译时根据构建标签替换
- **净增量网络字节**: 当前 `NetRxBps`/`NetTxBps` 存的是累计值，非速率（见 MetricSample 文档）

## 依赖

**本模块依赖**:
- `github.com/shirou/gopsutil/v3` — 跨平台系统信息采集（posix 使用）
- `github.com/go-ole/go-ole` / `github.com/lufia/plan9stats` — Windows COM/WMI（win 使用）

**被依赖**:
- 无（独立二进制，通过 HTTP 与服务端通信）

## 构建

```bash
# Linux
cd agent && GOOS=linux go build -o msmp-agent .

# Windows
cd agent && GOOS=windows go build -o msmp-agent.exe .

# macOS
cd agent && GOOS=darwin go build -o msmp-agent .
```
