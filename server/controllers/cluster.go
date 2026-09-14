package controllers

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"MSMP/server/clustering"
	"MSMP/server/db"
	"MSMP/server/models"
)

// remoteAddrHost 从 RemoteAddr（host:port）安全提取 host。
func remoteAddrHost(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

// normalizeNodeAddr 规范化节点地址：trim 空格、补 http 前缀、去尾斜杠。
func normalizeNodeAddr(raw string) string {
	addr := strings.TrimSpace(raw)
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	return strings.TrimRight(addr, "/")
}

// LoadClusterNodes 从数据库加载启用的集群节点到集群状态。
func LoadClusterNodes(state *clustering.ClusterState) {
	var nodes []models.ClusterNode
	db.DB.Where("enabled = ?", true).Find(&nodes)
	if len(nodes) == 0 {
		return
	}
	addrs := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if strings.TrimSpace(n.Address) != "" {
			addrs = append(addrs, strings.TrimSpace(n.Address))
		}
	}
	if len(addrs) > 0 {
		state.RefreshNodes(addrs)
	}
}

// ClusterNodesHandler GET/POST /api/cluster/nodes
func ClusterNodesHandler(w http.ResponseWriter, r *http.Request, state *clustering.ClusterState) {
	switch r.Method {
	case http.MethodGet:
		var nodes []models.ClusterNode
		db.DB.Order("id asc").Find(&nodes)
		list := make([]map[string]interface{}, 0, len(nodes))
		for _, n := range nodes {
			list = append(list, map[string]interface{}{
				"id":         n.ID,
				"name":       n.Name,
				"address":    n.Address,
				"enabled":    n.Enabled,
				"alive":      state.NodeAlive(n.Address),
				"created_at": n.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var req struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Address) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and address are required"})
			return
		}
		addr := normalizeNodeAddr(req.Address)
		var exists models.ClusterNode
		if err := db.DB.Where("address = ?", addr).First(&exists).Error; err == nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "node address already exists"})
			return
		}
		node := models.ClusterNode{Name: strings.TrimSpace(req.Name), Address: addr, Enabled: true}
		if err := db.DB.Create(&node).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create node"})
			return
		}
		LoadClusterNodes(state)
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"id": node.ID, "name": node.Name, "address": node.Address, "enabled": node.Enabled,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// ClusterNodeDetailHandler PUT/DELETE /api/cluster/nodes/{id}
func ClusterNodeDetailHandler(w http.ResponseWriter, r *http.Request, state *clustering.ClusterState) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/cluster/nodes/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var node models.ClusterNode
	if err := db.DB.First(&node, id).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name    string `json:"name"`
			Address string `json:"address"`
			Enabled *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		updates := map[string]interface{}{}
		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.Address != "" {
			updates["address"] = normalizeNodeAddr(req.Address)
		}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if err := db.DB.Model(&node).Updates(updates).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update node"})
			return
		}
		LoadClusterNodes(state)
		db.DB.First(&node, id)
		writeJSON(w, http.StatusOK, node)

	case http.MethodDelete:
		if err := db.DB.Delete(&node).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete node"})
			return
		}
		LoadClusterNodes(state)
		writeJSON(w, http.StatusOK, map[string]string{"deleted": "true"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
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
		"leader": state.LeaderAddress(),
		"mode":   state.Mode(),
		"ts":     time.Now().Unix(),
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
