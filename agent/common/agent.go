package common

// ...（前略）
func ReportAssets(serverURL string, info AgentInfo) error {
    data, _ := json.Marshal(info)
    _, err := http.Post(serverURL+"/api/agents/assets", "application/json", bytes.NewBuffer(data))
    return err
}

func MainLoop(serverURL string) {
    for {
        // 心跳
        Heartbeat(serverURL, os.Getenv("AGENT_UUID"))
        // 采集并上报资产，每5分钟一次
        if time.Now().Minute()%5 == 0 {
            assetInfo := CollectAssetInfo() // platform-specific function
            ReportAssets(serverURL, assetInfo)
        }
        // 拉取任务、执行
        time.Sleep(30 * time.Second)
    }
}