// Package services 提供可用性探测服务。
package services

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"
)

// ProbeResult 单次探测结果。
type ProbeResult struct {
	Up       bool   `json:"up"`
	Latency  int64  `json:"latency_ms"`
	Error    string `json:"error,omitempty"`
	Code     int    `json:"code,omitempty"`
	Issues   int    `json:"issues,omitempty"` // SSL: days until expiry
}

// ProbeHTTP 探测 HTTP/HTTPS 端点。
func ProbeHTTP(ctx context.Context, target string, timeoutSec, expectedCode int) ProbeResult {
	start := time.Now()
	client := &http.Client{
		Timeout:   time.Duration(timeoutSec) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return ProbeResult{Up: false, Latency: time.Since(start).Milliseconds(), Error: err.Error()}
	}
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ProbeResult{Up: false, Latency: latency, Error: err.Error()}
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	up := true
	if expectedCode > 0 && code != expectedCode {
		up = false
	}
	return ProbeResult{Up: up, Latency: latency, Code: code}
}

// ProbeTCP 探测 TCP 端口连通性。
func ProbeTCP(ctx context.Context, target string, timeoutSec int) ProbeResult {
	start := time.Now()
	dialer := net.Dialer{Timeout: time.Duration(timeoutSec) * time.Second}
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return ProbeResult{Up: false, Latency: time.Since(start).Milliseconds(), Error: "invalid target: " + target}
	}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ProbeResult{Up: false, Latency: latency, Error: err.Error()}
	}
	conn.Close()
	return ProbeResult{Up: true, Latency: latency}
}

// ProbeSSL 探测 SSL 证书有效期。
func ProbeSSL(ctx context.Context, target string, timeoutSec int) ProbeResult {
	start := time.Now()
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return ProbeResult{Up: false, Latency: time.Since(start).Milliseconds(), Error: "invalid target: " + target}
	}
	port := portStr
	if port == "" {
		port = "443"
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: time.Duration(timeoutSec) * time.Second}, "tcp",
		net.JoinHostPort(host, port), &tls.Config{InsecureSkipVerify: true})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ProbeResult{Up: false, Latency: latency, Error: err.Error()}
	}
	defer conn.Close()

	cert := conn.ConnectionState().PeerCertificates
	if len(cert) == 0 {
		return ProbeResult{Up: true, Latency: latency}
	}
	now := time.Now()
	expiry := cert[0].NotAfter
	days := int(expiry.Sub(now).Hours() / 24)
	issues := 0
	if days < 0 {
		issues = -days
	} else if days < 30 {
		issues = days
	}
	return ProbeResult{Up: true, Latency: latency, Issues: issues}
}

// NormalizeProbeTarget 规范化探测目标地址。
func NormalizeProbeTarget(target, proto string) string {
	target = strings.TrimSpace(target)
	if proto == "tcp" || proto == "ssl" {
		if !strings.Contains(target, ":") {
			defaultPort := "80"
			if proto == "ssl" {
				defaultPort = "443"
			}
			return target + ":" + defaultPort
		}
		return target
	}
	// HTTP/HTTPS
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}
	return target
}
