package main

import (
	"log"
	"os"

	"MSMP/agent/common"
	"MSMP/agent/win"
)

func main() {
	// 绑定平台实现
	common.CollectAssetInfoFull = win.CollectAssetInfoFull
	common.CollectMetrics = win.CollectMetrics

	serverURL := common.GetEnv("MSMP_SERVER_URL", "http://localhost:8080")
	agentUUID := common.GetEnv("AGENT_UUID", "")
	agentToken := common.GetEnv("AGENT_TOKEN", "")

	if agentUUID == "" {
		hostname, _ := os.Hostname()
		agentUUID = hostname
		log.Printf("AGENT_UUID not set, using hostname: %s", agentUUID)
	}

	log.Printf("MSMP Agent starting...")
	log.Printf("  Server: %s", serverURL)
	log.Printf("  UUID: %s", agentUUID)

	common.MainLoop(serverURL, agentUUID, agentToken)
}