// Package collectors contains the vSphere/ESXi collector implementation.
package collectors

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"MSMP/server/models"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
)

// VSphereChannel implements the Channel interface for vSphere/ESXi hosts.
type VSphereChannel struct{}

func (v *VSphereChannel) Type() string { return "vsphere" }

// parseAddress normalizes the address to a SDK URL.
func parseAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	return "https://" + addr + "/sdk"
}

// parseCredential extracts username:password from the credential secret.
func parseCredential(secret string) (string, string) {
	parts := strings.SplitN(secret, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return secret, ""
}

func (v *VSphereChannel) dial(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (*govmomi.Client, error) {
	secret, err := cred.Decrypt(b.Credential)
	if err != nil {
		return nil, fmt.Errorf("%s:decrypt credential: %w", StatusAuthFailed, err)
	}

	username, password := parseCredential(secret)
	endpoint := parseAddress(b.Address)

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("%s:invalid endpoint: %w", StatusUnreachable, err)
	}

	c, err := govmomi.NewClient(ctx, u, true) // skip cert verify for lab
	if err != nil {
		return nil, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}

	userInfo := url.UserPassword(username, password)
	if err := c.Login(ctx, userInfo); err != nil {
		return nil, fmt.Errorf("%s:%s", StatusAuthFailed, err.Error())
	}

	return c, nil
}

// Probe connects to vSphere and verifies the host is reachable.
func (v *VSphereChannel) Probe(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	c, err := v.dial(ctx, b, cred)
	if err != nil {
		pr, _ := classify(err)
		return pr, err
	}
	defer c.Logout(ctx)

	about := c.ServiceContent.About
	return ProbeResult{
		OK:   true,
		OS:   "vmware-esxi",
		Host: about.Name,
	}, nil
}

// Collect gathers metrics from vSphere.
func (v *VSphereChannel) Collect(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	c, err := v.dial(ctx, b, cred)
	if err != nil {
		return CollectResult{}, err
	}
	defer c.Logout(ctx)

	var result CollectResult

	metrics, err := v.collectHostMetrics(ctx, c)
	if err != nil {
		log.Printf("[VSphere] host metrics error: %v", err)
	} else {
		result.Metrics = metrics
	}

	result.Duration = time.Since(start)
	return result, nil
}

// collectHostMetrics gathers CPU and memory from ESXi hosts.
func (v *VSphereChannel) collectHostMetrics(ctx context.Context, c *govmomi.Client) (MetricDataLike, error) {
	var m MetricDataLike

	// Get root folder and datacenters
	root := c.Client.ServiceContent.RootFolder
	dcList, err := object.NewFolder(c.Client, root).Children(ctx)
	if err != nil {
		return m, fmt.Errorf("list datacenters: %w", err)
	}

	if len(dcList) == 0 {
		return m, fmt.Errorf("no datacenters found")
	}

	// Use first datacenter
	dc := dcList[0].(*object.Datacenter)

	// Get hosts folder
	folders, err := dc.Folders(ctx)
	if err != nil {
		return m, fmt.Errorf("get folders: %w", err)
	}
	if folders.HostFolder == nil {
		return m, fmt.Errorf("no hosts folder")
	}

	hosts, err := folders.HostFolder.Children(ctx)
	if err != nil {
		return m, fmt.Errorf("get hosts: %w", err)
	}

	if len(hosts) == 0 {
		return m, fmt.Errorf("no ESXi hosts found")
	}

	// Aggregate metrics
	var totalCPU uint64
	var totalMemUsed, totalMemTotal uint64
	var count int

	for _, h := range hosts {
		hs := h.(*object.HostSystem)
		var info mo.HostSystem
		err := hs.Properties(ctx, hs.Reference(), []string{"summary"}, &info)
		if err != nil {
			continue
		}
		s := info.Summary
		totalCPU += uint64(s.QuickStats.OverallCpuUsage)
		totalMemUsed += uint64(s.QuickStats.OverallMemoryUsage) * 1024 * 1024
		if s.Hardware != nil && s.Hardware.MemorySize > 0 {
			totalMemTotal += uint64(s.Hardware.MemorySize)
		}
		count++
	}

	if count > 0 {
		// CPU usage in MHz, assume 1000MHz per core for percentage
		m.CPUPercent = float64(totalCPU) / float64(count) / 10.0 // rough estimate
		if totalMemTotal > 0 {
			m.MemPercent = float64(totalMemUsed) / float64(totalMemTotal) * 100
		}
		m.MemUsed = totalMemUsed
		m.MemTotal = totalMemTotal
	}

	return m, nil
}
