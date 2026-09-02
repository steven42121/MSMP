package collectors

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"MSMP/server/models"
)

const promTimeout = 15 * time.Second

type PrometheusChannel struct {
	HTTP *http.Client
}

func (p *PrometheusChannel) Type() string { return "prometheus" }

func (p *PrometheusChannel) client() *http.Client {
	if p.HTTP != nil {
		return p.HTTP
	}
	return &http.Client{Timeout: promTimeout}
}

func (p *PrometheusChannel) scrape(ctx context.Context, b *models.ChannelBinding) ([]byte, error) {
	addr := strings.TrimRight(b.Address, "/")
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	url := addr + "/metrics"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	switch b.AuthMode {
	case "basic":
		req.SetBasicAuth(b.Username, b.Credential)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+b.Credential)
	}
	resp, err := p.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return nil, fmt.Errorf("%s:status %d", StatusAuthFailed, resp.StatusCode)
		}
		return nil, fmt.Errorf("%s:status %d", StatusUnreachable, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s:read body: %s", StatusParseError, err.Error())
	}
	return body, nil
}

func parsePromFloat(body []byte, name string) (float64, bool) {
	sc := bufio.NewScanner(strings.NewReader(string(body)))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		idx := strings.Index(line, " ")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		if key != name {
			continue
		}
		val := strings.TrimSpace(line[idx+1:])
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			continue
		}
		return f, true
	}
	return 0, false
}

func parsePromUint64(body []byte, name string) (uint64, bool) {
	f, ok := parsePromFloat(body, name)
	if !ok {
		return 0, false
	}
	return uint64(f), true
}

func parsePromCounterSum(body []byte, metric string, labelFilter string) (uint64, bool) {
	sc := bufio.NewScanner(strings.NewReader(string(body)))
	var total uint64
	found := false
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		idx := strings.Index(line, " ")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		if key != metric {
			continue
		}
		if labelFilter != "" {
			rest := line[idx+1:]
			brace := strings.Index(rest, "{")
			if brace >= 0 {
				endBrace := strings.Index(rest, "}")
				if endBrace < 0 {
					continue
				}
				labels := rest[brace+1 : endBrace]
				if !strings.Contains(labels, labelFilter) {
					continue
				}
			}
		}
		val := strings.TrimSpace(line[idx+1:])
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			continue
		}
		total += uint64(f)
		found = true
	}
	return total, found
}

func parsePromCPU(body []byte) float64 {
	idleSum, hasIdle := parsePromCounterSum(body, "node_cpu_seconds_total", "cpu=\"idle\"")
	totalSum, hasTotal := parsePromCounterSum(body, "node_cpu_seconds_total", "")
	if !hasIdle || !hasTotal || totalSum == 0 {
		return 0
	}
	idle := float64(idleSum) / float64(totalSum) * 100
	used := 100 - idle
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	return used
}

func (p *PrometheusChannel) Probe(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	body, err := p.scrape(ctx, b)
	if err != nil {
		return ProbeResult{Err: classifyPromErr(err)}, err
	}
	osStr := "unknown"
	if _, ok := parsePromFloat(body, "node_uname_info"); ok {
		osStr = "linux"
	} else if _, ok := parsePromFloat(body, "windows_os_version"); ok {
		osStr = "windows"
	}
	return ProbeResult{OK: true, OS: osStr}, nil
}

func (p *PrometheusChannel) Collect(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	body, err := p.scrape(ctx, b)
	if err != nil {
		return CollectResult{}, err
	}

	missing := []string{}

	cpu := parsePromCPU(body)

	memTotal, hasMemTotal := parsePromFloat(body, "node_memory_MemTotal_bytes")
	memAvail, hasMemAvail := parsePromFloat(body, "node_memory_MemAvailable_bytes")
	memPct := 0.0
	memUsedU64 := uint64(0)
	if hasMemTotal && memTotal > 0 {
		if hasMemAvail {
			memUsedU64 = uint64(memTotal - memAvail)
			memPct = float64(memUsedU64) / memTotal * 100
		} else {
			memPct = 0
		}
	} else {
		memTotal = 0
	}
	if !hasMemTotal {
		missing = append(missing, "mem_total", "mem_used")
	}

	diskUsed, hasDiskUsed := parsePromCounterSum(body, "node_filesystem_size_bytes", "")
	diskTotal, hasDiskTotal := parsePromCounterSum(body, "node_filesystem_avail_bytes", "")
	if !hasDiskUsed || !hasDiskTotal {
		missing = append(missing, "disk_used", "disk_total")
	}

	netRx, hasNetRx := parsePromCounterSum(body, "node_network_receive_bytes_total", "")
	netTx, hasNetTx := parsePromCounterSum(body, "node_network_transmit_bytes_total", "")
	if !hasNetRx || !hasNetTx {
		missing = append(missing, "net_rx_bps", "net_tx_bps")
	}

	load1, hasLoad1 := parsePromFloat(body, "node_load1")
	load5, hasLoad5 := parsePromFloat(body, "node_load5")
	load15, hasLoad15 := parsePromFloat(body, "node_load15")
	if !hasLoad1 || !hasLoad5 || !hasLoad15 {
		missing = append(missing, "load1", "load5", "load15")
	}

	uptimeSec, hasUptime := parsePromFloat(body, "node_time_seconds")
	bootTime, hasBoot := parsePromFloat(body, "node_boot_time_seconds")
	up := uint64(0)
	if hasUptime && hasBoot {
		up = uint64(uptimeSec - bootTime)
	} else {
		missing = append(missing, "uptime_sec")
	}

	return CollectResult{
		Metrics: MetricDataLike{
			CPUPercent: cpu,
			MemPercent: memPct,
			MemUsed:    memUsedU64,
			MemTotal:   uint64(memTotal),
			DiskUsed:   diskUsed,
			DiskTotal:  diskTotal,
			NetRxBps:   netRx,
			NetTxBps:   netTx,
			Load1:      load1,
			Load5:      load5,
			Load15:     load15,
			UptimeSec:  up,
		},
		Missing:  missing,
		Duration: time.Since(start),
	}, nil
}

func classifyPromErr(err error) string {
	msg := err.Error()
	for _, s := range []string{StatusUnreachable, StatusAuthFailed, StatusDenied, StatusUnsupported, StatusParseError} {
		if strings.HasPrefix(msg, s) {
			return s
		}
	}
	return StatusUnreachable
}
