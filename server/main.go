package main
// ...（前略）
import "MSMP/server/controllers"
// ...（前略）
func main() {
    log.Println("MSMP Server Starting...")
    http.HandleFunc("/api/agents/register", controllers.AgentRegisterHandler)
    http.HandleFunc("/api/agents/heartbeat", controllers.AgentHeartbeatHandler)
    http.HandleFunc("/api/agents/assets", controllers.AgentAssetReportHandler) // 新增
    log.Fatal(http.ListenAndServe(":8080", nil))
}