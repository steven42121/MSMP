package collectors

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"MSMP/server/models"

	"github.com/gosnmp/gosnmp"
)

const snmpTimeout = 15 * time.Second

type SNMPChannel struct{}

func (s *SNMPChannel) Type() string { return "snmp" }

var (
	oidSysUpTime       = ".1.3.6.1.2.1.1.3.0"
	oidCPUTotalLoad    = ".1.3.6.1.4.1.2021.11.11.0"
	oidMemTotalReal    = ".1.3.6.1.4.1.2021.4.5.0"
	oidMemAvailReal    = ".1.3.6.1.4.1.2021.4.6.0"
	oidNetIfInOctets   = ".1.3.6.1.2.1.2.2.1.10"
	oidNetIfOutOctets  = ".1.3.6.1.2.1.2.2.1.16"
)

func (s *SNMPChannel) connect(ctx context.Context, b *models.ChannelBinding) (*gosnmp.GoSNMP, error) {
	host, port := parseHostPort(b.Address)
	portNum := 161
	if p, err := strconv.Atoi(port); err == nil && p > 0 {
		portNum = p
	}
	g := &gosnmp.GoSNMP{
		Target:    host,
		Port:      uint16(portNum),
		Community: "public",
		Version:   gosnmp.Version2c,
		Timeout:   snmpTimeout,
		Context:   ctx,
	}
	switch b.AuthMode {
	case "community":
		g.Community = b.Username
	case "v3":
		g.Version = gosnmp.Version3
		g.SecurityModel = gosnmp.UserSecurityModel
		fields := strings.Fields(b.Credential)
		sp := &gosnmp.UsmSecurityParameters{}
		if len(fields) >= 1 {
			sp.UserName = fields[0]
		}
		if len(fields) >= 2 {
			sp.AuthenticationProtocol = gosnmp.SHA
			sp.AuthenticationPassphrase = fields[1]
		}
		if len(fields) >= 3 {
			sp.PrivacyProtocol = gosnmp.AES
			sp.PrivacyPassphrase = fields[2]
		}
		g.SecurityParameters = sp
	default:
		g.Community = "public"
	}
	if err := g.Connect(); err != nil {
		return nil, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	return g, nil
}

func (s *SNMPChannel) Probe(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	g, err := s.connect(ctx, b)
	if err != nil {
		return ProbeResult{Err: classifySnmpErr(err)}, err
	}
	defer g.Conn.Close()
	res, err := g.Get([]string{oidSysUpTime})
	if err != nil {
		return ProbeResult{Err: StatusUnreachable}, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	if len(res.Variables) == 0 || res.Variables[0].Value == nil {
		return ProbeResult{Err: StatusParseError}, fmt.Errorf("%s:empty response", StatusParseError)
	}
	return ProbeResult{OK: true, OS: "unknown"}, nil
}

func (s *SNMPChannel) Collect(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	g, err := s.connect(ctx, b)
	if err != nil {
		return CollectResult{}, err
	}
	defer g.Conn.Close()

	missing := []string{}

	cpuRes, err := g.Get([]string{oidCPUTotalLoad})
	cpuLoad := 0.0
	if err != nil {
		missing = append(missing, "cpu_percent")
	} else if len(cpuRes.Variables) > 0 {
		if v, ok := cpuRes.Variables[0].Value.(int32); ok {
			cpuLoad = float64(v) / 100.0
			if cpuLoad > 100 {
				cpuLoad = 100
			}
		}
	}

	memRes, err := g.Get([]string{oidMemTotalReal, oidMemAvailReal})
	memTotal, memUsed := uint64(0), uint64(0)
	memPct := 0.0
	if err != nil {
		missing = append(missing, "mem_total", "mem_used")
	} else if len(memRes.Variables) >= 2 {
		tVal, _ := memRes.Variables[0].Value.(int32)
		aVal, _ := memRes.Variables[1].Value.(int32)
		memTotal = uint64(tVal) * 1024
		memUsed = (uint64(tVal) - uint64(aVal)) * 1024
		if memTotal > 0 {
			memPct = float64(memUsed) / float64(memTotal) * 100
		}
	} else {
		missing = append(missing, "mem_total", "mem_used")
	}

	netRes, err := g.GetBulk([]string{oidNetIfInOctets, oidNetIfOutOctets}, 12, 12)
	netRx, netTx := uint64(0), uint64(0)
	if err != nil {
		missing = append(missing, "net_rx_bps", "net_tx_bps")
	} else {
		for i := 0; i < len(netRes.Variables); i += 2 {
			if i+1 < len(netRes.Variables) {
				if v, ok := netRes.Variables[i].Value.(int32); ok {
					netRx += uint64(v)
				}
				if v, ok := netRes.Variables[i+1].Value.(int32); ok {
					netTx += uint64(v)
				}
			}
		}
	}

	upRes, err := g.Get([]string{oidSysUpTime})
	upSec := uint64(0)
	if err != nil {
		missing = append(missing, "uptime_sec")
	} else if len(upRes.Variables) > 0 {
		if v, ok := upRes.Variables[0].Value.(gosnmp.SnmpPDU); ok {
			if f, e2 := strconv.ParseFloat(fmt.Sprintf("%v", v.Value), 64); e2 == nil {
				upSec = uint64(f / 100)
			}
		}
	}

	load1, load5, load15 := 0.0, 0.0, 0.0
	if cpuLoad > 0 {
		load1, load5, load15 = cpuLoad, cpuLoad, cpuLoad
	} else if len(missing) == 0 {
		missing = append(missing, "load1", "load5", "load15")
	}

	return CollectResult{
		Metrics: MetricDataLike{
			CPUPercent: cpuLoad,
			MemPercent: memPct,
			MemUsed:    memUsed,
			MemTotal:   memTotal,
			NetRxBps:   netRx,
			NetTxBps:   netTx,
			Load1:      load1,
			Load5:      load5,
			Load15:     load15,
			UptimeSec:  upSec,
		},
		Missing:  missing,
		Duration: time.Since(start),
	}, nil
}

func classifySnmpErr(err error) string {
	msg := err.Error()
	for _, s := range []string{StatusUnreachable, StatusAuthFailed, StatusDenied, StatusUnsupported, StatusParseError} {
		if strings.HasPrefix(msg, s) {
			return s
		}
	}
	return StatusUnreachable
}
