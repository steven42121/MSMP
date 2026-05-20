package controllers

import (
    "encoding/json"
    "net/http"
    "log"
)

type AssetInfo struct {
    Hostname string
    OS       string
    IP       string
    CPU      string
    Memory   string
    Disk     string
    UUID     string
}

func AgentAssetReportHandler(w http.ResponseWriter, r *http.Request) {
    var info AssetInfo
    err := json.NewDecoder(r.Body).Decode(&info)
    if err != nil {
        log.Println("Error parsing asset info:", err)
        http.Error(w, "Invalid data", http.StatusBadRequest)
        return
    }
    // TODO: 资产信息存库
    log.Printf("Received asset info: %+v\n", info)
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
}