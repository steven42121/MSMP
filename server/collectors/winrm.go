package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"MSMP/server/models"

	"github.com/masterzen/winrm"
)

const winrmTimeoutStr = "PT20S"

type WinRMChannel struct{}

func (w *WinRMChannel) Type() string { return "winrm" }

func parseHostPortAddr(addr string) (string, int) {
	if !strings.Contains(addr, ":") {
		return addr, 5985
	}
	parts := strings.Split(addr, ":")
	host := parts[0]
	port := 5985
	if len(parts) > 1 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}
	return host, port
}

func (w *WinRMChannel) connect(b *models.ChannelBinding) (*winrm.Client, error) {
	host, port := parseHostPortAddr(b.Address)
	endpoint := winrm.NewEndpoint(host, port, false, false, nil, nil, nil, 0)
	params := winrm.NewParameters(winrmTimeoutStr, "en-US", 153600)
	client, err := winrm.NewClientWithParameters(endpoint, b.Username, b.Credential, params)
	if err != nil {
		return nil, fmt.Errorf("%s:%s", StatusAuthFailed, err.Error())
	}
	return client, nil
}

func (w *WinRMChannel) Probe(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	secret, err := cred.Decrypt(b.Credential)
	if err != nil {
		return ProbeResult{Err: StatusAuthFailed}, err
	}
	client, err := w.connect(&models.ChannelBinding{Username: b.Username, Credential: secret})
	if err != nil {
		return ProbeResult{Err: classifyWinRM(err)}, err
	}
	defer func() { _ = client }()
	_, _, _, err = client.RunPSWithContext(ctx, "(Get-CimInstance -ClassName Win32_OperatingSystem).Version")
	if err != nil {
		return ProbeResult{Err: classifyWinRM(err)}, err
	}
	return ProbeResult{OK: true, OS: "windows"}, nil
}

func classifyWinRM(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "dial"):
		return StatusUnreachable
	case strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "unauthorized"):
		return StatusAuthFailed
	default:
		return StatusUnsupported
	}
}

func (w *WinRMChannel) Collect(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	secret, err := cred.Decrypt(b.Credential)
	if err != nil {
		return CollectResult{}, err
	}
	client, err := w.connect(&models.ChannelBinding{Username: b.Username, Credential: secret})
	if err != nil {
		return CollectResult{}, err
	}
	defer func() { _ = client }()

	psScript := `
$cpu = Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average | Select-Object -ExpandProperty Average
$os = Get-CimInstance Win32_OperatingSystem
$memTotal = $os.TotalVisibleMemorySize
$memFree = $os.FreePhysicalMemory
$memUsed = $memTotal - $memFree
$memPct = [math]::Round(($memUsed / $memTotal) * 100, 2)
$disk = Get-CimInstance Win32_LogicalDisk -Filter "DeviceID='C:'"
$diskUsed = [math]::Round($disk.Size - $disk.FreeSpace, 0)
$diskTotal = [math]::Round($disk.Size, 0)
$uptime = (Get-Date) - $os.LastBootUpTime
$uptimeSec = [math]::Round($uptime.TotalSeconds, 0)
$json = @{
  cpu_percent = [math]::Round($cpu, 2)
  mem_percent = $memPct
  mem_used = [math]::Round($memUsed, 0)
  mem_total = [math]::Round($memTotal, 0)
  disk_used = [math]::Round($diskUsed, 0)
  disk_total = [math]::Round($diskTotal, 0)
  uptime_sec = [math]::Round($uptimeSec, 0)
} | ConvertTo-Json
Write-Output $json
`
	out, _, _, err := client.RunPSWithContext(ctx, psScript)
	if err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}

	var raw struct {
		CPUPercent float64 `json:"cpu_percent"`
		MemPercent float64 `json:"mem_percent"`
		MemUsed    float64 `json:"mem_used"`
		MemTotal   float64 `json:"mem_total"`
		DiskUsed   float64 `json:"disk_used"`
		DiskTotal  float64 `json:"disk_total"`
		UptimeSec  float64 `json:"uptime_sec"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}

	return CollectResult{
		Metrics: MetricDataLike{
			CPUPercent: raw.CPUPercent,
			MemPercent: raw.MemPercent,
			MemUsed:    uint64(raw.MemUsed),
			MemTotal:   uint64(raw.MemTotal),
			DiskUsed:   uint64(raw.DiskUsed),
			DiskTotal:  uint64(raw.DiskTotal),
			UptimeSec:  uint64(raw.UptimeSec),
		},
		Missing:  []string{"net_rx_bps", "net_tx_bps", "load1", "load5", "load15"},
		Duration: time.Since(start),
	}, nil
}
