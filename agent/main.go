package main

import (
	"log"
	"os"

	"MSMP/agent/common"
	_ "MSMP/agent/platform"
)

func main() {
	serverURLs := common.GetEnv("MSMP_SERVER_URLS", common.GetEnv("MSMP_SERVER_URL", "http://localhost:8080"))
	agentUUID := common.GetEnv("AGENT_UUID", "")
	agentToken := common.GetEnv("AGENT_TOKEN", "")

	if agentUUID == "" {
		hostname, _ := os.Hostname()
		agentUUID = hostname
		log.Printf("AGENT_UUID not set, using hostname: %s", agentUUID)
	}

	router := common.NewClusterRouter(serverURLs)

	log.Printf("MSMP Agent starting...")
	log.Printf("  Server(s): %v", router.Nodes())
	log.Printf("  UUID: %s", agentUUID)

	common.MainLoop(router, agentUUID, agentToken)
}

