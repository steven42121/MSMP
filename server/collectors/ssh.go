package collectors

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"MSMP/server/models"

	"golang.org/x/crypto/ssh"
)

const sshDialTimeout = 15 * time.Second

type SSHChannel struct{}

func (s *SSHChannel) Type() string { return "ssh" }

func parseHostPort(addr string) (string, string) {
	if !strings.Contains(addr, ":") {
		return addr, "22"
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, "22"
	}
	return host, port
}

func (s *SSHChannel) dial(ctx context.Context, b *models.ChannelBinding, secret string) (*ssh.Client, error) {
	host, port := parseHostPort(b.Address)
	auth, err := buildSSHAuth(b.AuthMode, secret)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:            b.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshDialTimeout,
	}
	d := net.Dialer{Timeout: sshDialTimeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, fmt.Errorf("%s:%s", StatusUnreachable, err.Error())
	}
	c, chs, reqs, err := ssh.NewClientConn(conn, net.JoinHostPort(host, port), cfg)
	if err != nil {
		return nil, fmt.Errorf("%s:%s", StatusAuthFailed, err.Error())
	}
	return ssh.NewClient(c, chs, reqs), nil
}

func buildSSHAuth(authMode, secret string) ([]ssh.AuthMethod, error) {
	switch authMode {
	case "password":
		return []ssh.AuthMethod{ssh.Password(secret)}, nil
	case "private_key", "generated_key":
		signer, err := ssh.ParsePrivateKey([]byte(secret))
		if err != nil {
			return nil, fmt.Errorf("%s:%s", StatusAuthFailed, err.Error())
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	default:
		return nil, fmt.Errorf("%s:unknown auth mode %s", StatusAuthFailed, authMode)
	}
}

func (s *SSHChannel) run(ctx context.Context, client *ssh.Client, cmd string) (string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	done := make(chan struct {
		out string
		err error
	}, 1)
	go func() {
		out, err := sess.CombinedOutput(cmd)
		done <- struct {
			out string
			err error
		}{string(out), err}
	}()
	select {
	case r := <-done:
		return r.out, r.err
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		return "", ctx.Err()
	}
}

func (s *SSHChannel) Probe(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (ProbeResult, error) {
	secret, err := cred.Decrypt(b.Credential)
	if err != nil {
		return ProbeResult{Err: StatusAuthFailed}, err
	}
	client, err := s.dial(ctx, b, secret)
	if err != nil {
		pr, _ := classify(err)
		return pr, err
	}
	defer client.Close()
	osOut, _ := s.run(ctx, client, "uname -s")
	hOut, _ := s.run(ctx, client, "hostname")
	osStr := strings.TrimSpace(string(osOut))
	if osStr == "" {
		osStr = "linux"
	}
	if !isLinux(osStr) {
		return ProbeResult{OK: false, OS: osStr, Host: strings.TrimSpace(string(hOut)), Err: StatusUnsupported}, fmt.Errorf("%s:non-linux os %s", StatusUnsupported, osStr)
	}
	return ProbeResult{OK: true, OS: osStr, Host: strings.TrimSpace(string(hOut))}, nil
}

func isLinux(osStr string) bool {
	o := strings.ToLower(strings.TrimSpace(osStr))
	return o == "linux" || o == "" || strings.Contains(o, "linux")
}

func (s *SSHChannel) Collect(ctx context.Context, b *models.ChannelBinding, cred CredentialProvider) (CollectResult, error) {
	start := time.Now()
	secret, err := cred.Decrypt(b.Credential)
	if err != nil {
		return CollectResult{}, err
	}
	client, err := s.dial(ctx, b, secret)
	if err != nil {
		return CollectResult{}, err
	}
	defer client.Close()

	cpuOut1, err := s.run(ctx, client, "cat /proc/stat")
	if err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	memOut, err := s.run(ctx, client, "cat /proc/meminfo")
	if err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	dfOut, err := s.run(ctx, client, "df -P")
	if err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	netOut, err := s.run(ctx, client, "cat /proc/net/dev")
	if err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	loadOut, err := s.run(ctx, client, "cat /proc/loadavg")
	if err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	upOut, err := s.run(ctx, client, "cat /proc/uptime")
	if err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}
	cpuOut2, err := s.run(ctx, client, "cat /proc/stat")
	if err != nil {
		return CollectResult{}, fmt.Errorf("%s:%s", StatusParseError, err.Error())
	}

	cpu := parseCPUStat(cpuOut1, cpuOut2)
	mem, _ := parseMeminfo(memOut)
	diskUsed, diskTotal := parseDisk(dfOut)
	netRx, netTx := parseNetDev(netOut)
	l1, l5, l15 := parseLoadavg(loadOut)
	up := parseUptime(upOut)

	memPct := 0.0
	if mem.Total > 0 {
		memPct = float64(mem.Used) / float64(mem.Total) * 100
	}
	diskPct := 0.0
	_ = diskPct

	return CollectResult{
		Metrics: MetricDataLike{
			CPUPercent: cpu,
			MemPercent: memPct,
			MemUsed:    mem.Used,
			MemTotal:   mem.Total,
			DiskUsed:   diskUsed,
			DiskTotal:  diskTotal,
			NetRxBps:   netRx,
			NetTxBps:   netTx,
			Load1:      l1,
			Load5:      l5,
			Load15:     l15,
			UptimeSec:  up,
		},
		Duration: time.Since(start),
	}, nil
}

func classify(err error) (ProbeResult, error) {
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, StatusUnreachable):
		return ProbeResult{Err: StatusUnreachable}, err
	case strings.HasPrefix(msg, StatusAuthFailed):
		return ProbeResult{Err: StatusAuthFailed}, err
	default:
		return ProbeResult{Err: StatusUnreachable}, err
	}
}

// parseCPUStat computes overall CPU usage percent from two /proc/stat snapshots.
func parseCPUStat(snap1, snap2 string) float64 {
	idle1, total1 := cpuAggregate(snap1)
	idle2, total2 := cpuAggregate(snap2)
	dt := total2 - total1
	di := idle2 - idle1
	if dt <= 0 {
		return 0
	}
	used := (float64(dt-di) / float64(dt)) * 100
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	return used
}

// cpuAggregate parses the first "cpu " aggregate line, returns (idle, total) in jiffies.
func cpuAggregate(stat string) (idle, total uint64) {
	sc := bufio.NewScanner(strings.NewReader(stat))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0
		}
		// fields[1..] = user nice system idle iowait irq softirq steal ...
		var vals []uint64
		for _, f := range fields[1:] {
			n, _ := strconv.ParseUint(f, 10, 64)
			vals = append(vals, n)
		}
		idle = vals[3]
		if len(vals) > 4 {
			idle += vals[4] // iowait
		}
		for _, v := range vals {
			total += v
		}
		return idle, total
	}
	return 0, 0
}

type memInfo struct {
	Total uint64
	Used  uint64
}

func parseMeminfo(meminfo string) (memInfo, bool) {
	var total, avail uint64
	sc := bufio.NewScanner(strings.NewReader(meminfo))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total = parseKBValue(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			avail = parseKBValue(line)
		}
	}
	if total == 0 {
		return memInfo{}, false
	}
	used := total - avail
	return memInfo{Total: total * 1024, Used: used * 1024}, true
}

func parseKBValue(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	n, _ := strconv.ParseUint(fields[1], 10, 64)
	return n
}

// parseDisk sums used and total across non-pseudo filesystems (df -P).
func parseDisk(dfOut string) (used, total uint64) {
	sc := bufio.NewScanner(strings.NewReader(dfOut))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "Filesystem") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mnt := fields[5]
		if isPseudoMount(mnt) {
			continue
		}
		u, _ := strconv.ParseUint(fields[2], 10, 64) // Used (1K-blocks)
		t, _ := strconv.ParseUint(fields[1], 10, 64) // 1K-blocks
		used += u * 1024
		total += t * 1024
	}
	return used, total
}

func isPseudoMount(mnt string) bool {
	for _, p := range []string{"/dev", "/sys", "/proc", "/run", "/snap", "/var/lib/docker"} {
		if mnt == p || strings.HasPrefix(mnt, p+"/") {
			return true
		}
	}
	return strings.HasPrefix(mnt, "/var/lib/")
}

// parseNetDev returns cumulative rx/tx bytes from /proc/net/dev.
func parseNetDev(out string) (rx, tx uint64) {
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:idx])
		if iface == "lo" {
			continue
		}
		rest := strings.Fields(line[idx+1:])
		if len(rest) < 9 {
			continue
		}
		r, _ := strconv.ParseUint(rest[0], 10, 64) // rx bytes
		t, _ := strconv.ParseUint(rest[8], 10, 64) // tx bytes
		rx += r
		tx += t
	}
	return rx, tx
}

// parseLoadavg parses /proc/loadavg, returns load1/5/15.
func parseLoadavg(out string) (l1, l5, l15 float64) {
	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) < 3 {
		return 0, 0, 0
	}
	l1, _ = strconv.ParseFloat(fields[0], 64)
	l5, _ = strconv.ParseFloat(fields[1], 64)
	l15, _ = strconv.ParseFloat(fields[2], 64)
	return l1, l5, l15
}

// parseUptime parses /proc/uptime, returns integer seconds.
func parseUptime(out string) uint64 {
	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) < 1 {
		return 0
	}
	f, _ := strconv.ParseFloat(fields[0], 64)
	return uint64(f)
}
