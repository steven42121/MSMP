package clustering

import (
	"testing"

	"MSMP/server/config"
)

func makeState(advertise string, nodes []string) *ClusterState {
	cfg := &config.Config{}
	cfg.Server.Addr = ":8080"
	cfg.Server.AdvertiseAddr = advertise
	cfg.Server.Nodes = nodes
	return NewClusterState(cfg)
}

// TestLeaderElection 验证多节点下按通告地址字典序选举唯一 leader。
func TestLeaderElection(t *testing.T) {
	addrs := []string{"http://node-1:8080", "http://node-2:8080", "http://node-3:8080"}
	states := make([]*ClusterState, 0, len(addrs))
	for _, a := range addrs {
		states = append(states, makeState(a, addrs))
	}

	// 模拟节点间心跳互认
	for i, s := range states {
		for j, a := range addrs {
			if i != j {
				s.RegisterNode(a, "node-"+string(rune('1'+j)))
			}
		}
	}

	// node-1 地址字典序最小，应成为 leader
	if !states[0].IsLeader() {
		t.Fatalf("node-1 should be leader")
	}
	if states[1].IsLeader() {
		t.Fatalf("node-2 should not be leader")
	}
	if states[2].IsLeader() {
		t.Fatalf("node-3 should not be leader")
	}
	if got := states[0].LeaderAddress(); got != "http://node-1:8080" {
		t.Fatalf("leader address = %q, want node-1", got)
	}
}

// TestAdvertiseAddrFallback 验证未配置 advertise_addr 时回退监听地址。
func TestAdvertiseAddrFallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Addr = ":8080"
	cfg.Server.Nodes = []string{}
	s := NewClusterState(cfg)
	if s.myAddress != ":8080" {
		t.Fatalf("myAddress = %q, want :8080 fallback", s.myAddress)
	}
}