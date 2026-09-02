package clustering

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"MSMP/server/config"

	"github.com/google/uuid"
)

const (
	heartbeatInterval = 10 * time.Second
	deadThreshold     = 30 * time.Second // 3 × heartbeatInterval
)

type NodeInfo struct {
	Address    string    `json:"address"`
	NodeID     string    `json:"node_id"`
	Alive      bool      `json:"alive"`
	LastPingAt time.Time `json:"last_ping_at"`
	FailCount  int       `json:"-"`
}

type ClusterState struct {
	mu           sync.RWMutex
	myAddress    string
	myNodeID     string
	nodes        map[string]*NodeInfo
	knownNodes   []string
	initializing bool
}

func NewClusterState(cfg *config.Config) *ClusterState {
	nodeID := cfg.Server.NodeID
	if nodeID == "" {
		nodeID = "node-" + uuid.New().String()[:8]
	}
	cs := &ClusterState{
		myAddress:  cfg.Server.Addr,
		myNodeID:   nodeID,
		nodes:      make(map[string]*NodeInfo),
		knownNodes: cfg.Server.Nodes,
	}
	// 将本机加入节点列表（如果不已经在列表中）
	if !contains(cs.knownNodes, cs.myAddress) {
		cs.knownNodes = append(cs.knownNodes, cs.myAddress)
	}
	cs.initializing = true
	log.Printf("[cluster] initialized: my_address=%s, nodes=%v", cs.myAddress, cs.knownNodes)
	return cs
}

func contains(list []string, target string) bool {
	for _, s := range list {
		if strings.TrimRight(s, "/") == strings.TrimRight(target, "/") {
			return true
		}
	}
	return false
}

// RegisterNode 收到心跳时更新节点状态。
func (c *ClusterState) RegisterNode(address, nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	addr := strings.TrimRight(address, "/")
	now := time.Now()
	if existing, ok := c.nodes[addr]; ok {
		existing.Alive = true
		existing.LastPingAt = now
		existing.NodeID = nodeID
		existing.FailCount = 0
	} else {
		c.nodes[addr] = &NodeInfo{
			Address:    addr,
			NodeID:     nodeID,
			Alive:      true,
			LastPingAt: now,
		}
	}
	log.Printf("[cluster] node registered: %s (id=%s)", addr, nodeID)
}

// DeregisterDeadNodes 移除超时未收到心跳的节点。
func (c *ClusterState) DeregisterDeadNodes() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for addr, node := range c.nodes {
		if now.Sub(node.LastPingAt) > deadThreshold && addr != c.myAddress {
			log.Printf("[cluster] node marked unreachable: %s", addr)
			node.Alive = false
		}
	}
}

// GetHealthyNodes 返回所有存活节点地址列表（含本机）。
func (c *ClusterState) GetHealthyNodes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var result []string
	for addr, node := range c.nodes {
		if node.Alive {
			result = append(result, addr)
		}
	}
	if !contains(result, c.myAddress) {
		result = append(result, c.myAddress)
	}
	return result
}

// IsLeader 判断本机是否为 leader。
// 规则：所有存活节点按地址字典序排序，最小的为 leader。
func (c *ClusterState) IsLeader() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	healthy := c.getSortedHealthyNodes()
	if len(healthy) == 0 {
		return true
	}
	return healthy[0] == c.myAddress
}

func (c *ClusterState) getSortedHealthyNodes() []string {
	healthy := make([]string, 0, len(c.nodes)+1)
	for addr, node := range c.nodes {
		if node.Alive {
			healthy = append(healthy, addr)
		}
	}
	if !contains(healthy, c.myAddress) {
		healthy = append(healthy, c.myAddress)
	}
	sort.Strings(healthy)
	return healthy
}

// LeaderAddress 返回当前 leader 的地址。
func (c *ClusterState) LeaderAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	healthy := c.getSortedHealthyNodes()
	if len(healthy) > 0 {
		return healthy[0]
	}
	return c.myAddress
}

// Mode 返回当前集群模式（standalone 或 cluster）。
func (c *ClusterState) Mode() string {
	if len(c.knownNodes) > 1 {
		return "cluster"
	}
	return "standalone"
}

// PublishInfo 写入集群状态到 HTTP 响应。
func (c *ClusterState) PublishInfo(w http.ResponseWriter) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodeList := make([]map[string]interface{}, 0, len(c.nodes)+1)
	for addr, node := range c.nodes {
		nodeList = append(nodeList, map[string]interface{}{
			"address":     addr,
			"node_id":     node.NodeID,
			"alive":       node.Alive,
			"last_ping":   node.LastPingAt.Format(time.RFC3339),
		})
	}
	// 确保本机在列表中
	found := false
	for _, n := range nodeList {
		if n["address"] == c.myAddress {
			found = true
			break
		}
	}
	if !found {
		nodeList = append(nodeList, map[string]interface{}{
			"address":   c.myAddress,
			"node_id":   c.myNodeID,
			"alive":     true,
			"last_ping": time.Now().Format(time.RFC3339),
		})
	}

	resp := map[string]interface{}{
		"mode":    c.Mode(),
		"node_id": c.myNodeID,
		"address": c.myAddress,
		"leader":  c.LeaderAddress(),
		"nodes":   nodeList,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// PublishLeader 仅返回 leader 地址。
func (c *ClusterState) PublishLeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"leader": c.LeaderAddress()})
}

// StartHeartbeatLoop 在 follower 节点上启动，定期向其他节点发送心跳。
func (c *ClusterState) StartHeartbeatLoop() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for range ticker.C {
		c.sendHeartbeats()
		c.DeregisterDeadNodes()
	}
}

func (c *ClusterState) sendHeartbeats() {
	healthy := c.GetHealthyNodes()
	for _, target := range healthy {
		if target == c.myAddress {
			continue
		}
		go c.pingNode(target)
	}
}

func (c *ClusterState) pingNode(target string) {
	url := strings.TrimRight(target, "/") + "/api/cluster/ping"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[cluster] heartbeat to %s failed: %v", target, err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[cluster] heartbeat to %s returned %d", target, resp.StatusCode)
	}
}
