// Package collectors contains the Proxmox VE collector implementation.
package collectors

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"MSMP/server/models"
	"MSMP/server/services"
)

// PVEChannel implements the Channel interface for Proxmox VE hosts.
type PVEChannel struct{}

func (p *PVEChannel) Type() string { return "pve" }

// newPVEClientFromBinding 解密凭据并建立 PVE 客户端连接。
func newPVEClientFromBinding(b *models.ChannelBinding, cred CredentialProvider) (*services.PVEClient, error) {
	secret, err := cred.Decrypt(b.Credential)
	if err != nil {
		return nil, fmt.Errorf("%s:decrypt credential: %w", StatusAuthFailed, err)
	}
	username, password := ParsePVECredential(secret)
	client, err := services.NewPVEClient(context.Background(), b.Address, username, password)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// ParsePVECredential extracts username:password. Username may include realm
// (e.g. root@pam); a bare username defaults to realm "pam".
func ParsePVECredential(secret string) (string, string) {
	parts := strings.SplitN(secret, ":", 2)
	username, password := secret, ""
	if len(parts) == 2 {
		username, password = parts[0], parts[1]
	}
	if username != "" && !strings.Contains(username, "@") {
		username = username + "@pam"
	}
	return username, password
}

func (p *PVEChannel) Probe(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	client, err := newPVEClientFromBinding(b, cred)
	if err != nil {
		pr, _ := classify(err)
		return pr, err
	}
	defer client.Logout()

	info, err := client.Version()
	if err != nil {
		pr, _ := classify(err)
		return pr, err
	}
	return ProbeResult{
		OK:   true,
		OS:   "proxmox-ve",
		Host: info.Version,
	}, nil
}

// Collect gathers aggregated CPU/memory/load metrics across all PVE nodes.
func (p *PVEChannel) Collect(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	client, err := newPVEClientFromBinding(b, cred)
	if err != nil {
		return CollectResult{}, err
	}
	defer client.Logout()

	var result CollectResult

	metrics, err := p.collectNodeMetrics(client)
	if err != nil {
		log.Printf("[PVE] node metrics error: %v", err)
	} else {
		result.Metrics = metrics
	}

	result.Duration = time.Since(start)
	return result, nil
}

// collectNodeMetrics aggregates CPU/memory/load across all online nodes.
func (p *PVEChannel) collectNodeMetrics(client *services.PVEClient) (MetricDataLike, error) {
	var m MetricDataLike

	nodes, err := client.Nodes()
	if err != nil {
		return m, fmt.Errorf("list nodes: %w", err)
	}

	var count int
	var cpuSum float64
	var memUsed, memTotal uint64
	var load1, load5, load15 float64
	var maxUptime uint64

	for _, node := range nodes {
		if node.Status != "online" {
			continue
		}
		status, err := client.NodeStatus(node.Node)
		if err != nil {
			continue
		}
		cpuSum += status.CPU
		memUsed += status.Memory.Used
		memTotal += status.Memory.Total
		load1 += status.LoadAvg[0]
		if len(status.LoadAvg) > 1 {
			load5 += status.LoadAvg[1]
		}
		if len(status.LoadAvg) > 2 {
			load15 += status.LoadAvg[2]
		}
		if status.Uptime > maxUptime {
			maxUptime = status.Uptime
		}
		count++
	}

	if count > 0 {
		m.CPUPercent = cpuSum / float64(count) * 100
		if memTotal > 0 {
			m.MemPercent = float64(memUsed) / float64(memTotal) * 100
		}
		m.MemUsed = memUsed
		m.MemTotal = memTotal
		m.Load1 = load1 / float64(count)
		m.Load5 = load5 / float64(count)
		m.Load15 = load15 / float64(count)
		m.UptimeSec = maxUptime
	}

	return m, nil
}
