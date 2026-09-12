package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// ClusterRouter 实现 Agent 对多个后端节点的轮询 + 故障转移。
type ClusterRouter struct {
	nodes         []string
	index         int
	failCount     map[int]int
	successCnt    map[int]int
	disabledUntil map[int]time.Time
}

func NewClusterRouter(serverURLs string) *ClusterRouter {
	raw := strings.Split(strings.TrimSpace(serverURLs), ",")
	nodes := make([]string, 0, len(raw))
	for _, u := range raw {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		u = strings.TrimRight(u, "/")
		if !strings.HasPrefix(u, "http") {
			u = "http://" + u
		}
		nodes = append(nodes, u)
	}
	if len(nodes) == 0 {
		nodes = append(nodes, "http://localhost:8080")
	}
	r := &ClusterRouter{
		nodes:         nodes,
		failCount:     make(map[int]int),
		successCnt:    make(map[int]int),
		disabledUntil: make(map[int]time.Time),
	}
	log.Printf("[agent] cluster router initialized: %d node(s): %v", len(nodes), nodes)
	return r
}

// NextNode 返回下一个可用节点 URL。
// 连续失败 3 次的节点暂时跳过，60 秒后重试。
func (r *ClusterRouter) NextNode() string {
	now := time.Now()
	var fallback = -1
	for i := 0; i < len(r.nodes); i++ {
		idx := (r.index + i) % len(r.nodes)
		if r.failCount[idx] < 3 {
			r.index = (idx + 1) % len(r.nodes)
			return r.nodes[idx]
		}
		if until, ok := r.disabledUntil[idx]; ok && now.After(until) {
			// 熔断窗口已过，重置并复用该节点
			r.failCount[idx] = 0
			r.index = (idx + 1) % len(r.nodes)
			return r.nodes[idx]
		}
		if fallback < 0 {
			fallback = idx
		}
	}
	// 所有节点均在熔断中：返回第一个节点兜底（请求可能失败，但不会饿死）
	r.index = (fallback + 1) % len(r.nodes)
	return r.nodes[fallback]
}

// RecordSuccess 标记当前节点上报成功。
func (r *ClusterRouter) RecordSuccess() {
	idx := r.index
	if idx >= 0 && idx < len(r.nodes) {
		r.failCount[idx] = 0
		r.successCnt[idx]++
	}
}

// RecordFailure 标记当前节点上报失败。
func (r *ClusterRouter) RecordFailure() {
	idx := r.index
	if idx >= 0 && idx < len(r.nodes) {
		r.failCount[idx]++
		log.Printf("[agent] node %s failed (%d/3)", r.nodes[idx], r.failCount[idx])
		if r.failCount[idx] >= 3 {
			r.disabledUntil[idx] = time.Now().Add(60 * time.Second)
			log.Printf("[agent] node %s circuit-breaked for 60s", r.nodes[idx])
		}
	}
}

// SelectNext 切换到下一个节点（调用方在报错时调用）。
func (r *ClusterRouter) SelectNext() string {
	r.index = (r.index + 1) % len(r.nodes)
	return r.nodes[r.index]
}

// NodeCount 返回节点数量。
func (r *ClusterRouter) NodeCount() int {
	return len(r.nodes)
}

// Nodes 返回节点列表。
func (r *ClusterRouter) Nodes() []string {
	return r.nodes
}

// RoundRobinURL 将 path 拼接到当前轮询到的节点 URL。
func (r *ClusterRouter) RoundRobinURL(path string) string {
	return r.NextNode() + path
}

// PostJSON 向当前节点发送 JSON POST 请求，失败时自动切换节点重试。
func (r *ClusterRouter) PostJSON(path string, body interface{}) error {
	url := r.RoundRobinURL(path)
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := doPost(url, data); err != nil {
		r.RecordFailure()
		return err
	}
	r.RecordSuccess()
	return nil
}

func doPost(url string, body []byte) error {
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return nil
}
