package win

import (
    "os"
    "MSMP/agent/common"
)

func CollectAssetInfo() common.AgentInfo {
    hostname, _ := os.Hostname()
    // IP需要调用网络库，CPU/Memory建议用 github.com/shirou/gopsutil
    ip := "127.0.0.1" // TODO: 实际采集
    osName := "Windows"
    uuid := hostname // TODO: 实际采集UUID

    return common.AgentInfo{
        Hostname: hostname,
        OS: osName,
        IP: ip,
        UUID: uuid,
    }
}