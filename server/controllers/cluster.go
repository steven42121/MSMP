package controllers

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"MSMP/server/clustering"
)

// remoteAddrHost 从 RemoteAddr（host:port）安全提取 host。
func remoteAddrHost(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

// ClusterPingHandler POST /api/cluster/ping
// 接收心跳，更新节点状态，返回当前 leader 地址。
func ClusterPingHandler(w http.ResponseWriter, r *http.Request, state *clustering.ClusterState) {
	var body struct {
		NodeID    string `json:"node_id"`
		Address   string `json:"address"`
		StartedAt string `json:"started_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// 兼容无 body 的心跳请求
		body.NodeID = ""
		body.Address = remoteAddrHost(r.RemoteAddr)
	}
	if body.Address == "" {
		body.Address = remoteAddrHost(r.RemoteAddr)
	}
	state.RegisterNode(body.Address, body.NodeID)

	resp := map[string]interface{}{
		"leader":   state.LeaderAddress(),
		"mode":     state.Mode(),
		"ts":       time.Now().Unix(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ClusterInfoHandler GET /api/cluster/info
func ClusterInfoHandler(w http.ResponseWriter, r *http.Request, state *clustering.ClusterState) {
	state.PublishInfo(w)
}

// ClusterLeaderHandler GET /api/cluster/leader
func ClusterLeaderHandler(w http.ResponseWriter, r *http.Request, state *clustering.ClusterState) {
	state.PublishLeader(w)
}
