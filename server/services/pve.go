// Package services 提供 Proxmox VE 管理能力。
package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// PVEClient 封装 Proxmox VE REST API（/api2/json）。
type PVEClient struct {
	baseURL  string
	username string
	password string
	ticket   string
	csrf     string
	http     *http.Client
	mu       sync.Mutex
}

// PVEVersionResponse 版本信息。
type PVEVersionResponse struct {
	Version    string `json:"version"`
	Release    string `json:"release"`
	RepoVersion string `json:"repoversion"`
}

// PVENode 集群节点摘要。
type PVENode struct {
	Node   string `json:"node"`
	Status string `json:"status"`
}

// PVEMemoryStatus 内存使用。
type PVEMemoryStatus struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

// PVENodeStatus 节点状态详情。
type PVENodeStatus struct {
	CPU     float64         `json:"cpu"`
	Memory  PVEMemoryStatus `json:"memory"`
	Uptime  uint64          `json:"uptime"`
	LoadAvg []float64       `json:"loadavg"`
	PVEVersion string       `json:"pveversion"`
}

// PVEMount 根分区 / 挂载信息。
type PVEMount struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

// PVEGuest 虚拟机 / LXC 容器信息。
type PVEGuest struct {
	VMID    int    `json:"vmid"`
	Node    string `json:"node"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	GuestType string `json:"guest_type"` // qemu | lxc
	CPUs    int    `json:"cpus"`
	MaxMem  uint64 `json:"maxmem"`
	Mem     uint64 `json:"mem"`
	MaxDisk uint64 `json:"maxdisk"`
	Disk    uint64 `json:"disk"`
	Uptime  uint64 `json:"uptime"`
	Template bool  `json:"template"`
	NetIn   uint64 `json:"netin"`
	NetOut  uint64 `json:"netout"`
}

// PVEStorage 数据存储信息。
type PVEStorage struct {
	Storage  string `json:"storage"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	Total    uint64 `json:"total"`
	Used     uint64 `json:"used"`
	Avail    uint64 `json:"avail"`
	Active   int    `json:"active"`
	Enabled  int    `json:"enabled"`
	Node     string `json:"-"`
}

// parsePVEAddress 规范化地址为 PVE API 入口（https://host:8006）。
func parsePVEAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "https://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		return addr
	}
	if u.Port() == "" {
		u.Host = u.Host + ":8006"
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String()
}

// NewPVEClient 创建客户端并登录。
func NewPVEClient(ctx context.Context, address, username, password string) (*PVEClient, error) {
	if username == "" {
		username = "root@pam"
	} else if !strings.Contains(username, "@") {
		username = username + "@pam"
	}

	c := &PVEClient{
		baseURL:  parsePVEAddress(address),
		username: username,
		password: password,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// login 获取 API ticket。
func (c *PVEClient) login(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api2/json/access/ticket", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("创建登录请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("unreachable:连接 PVE 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("auth_failed:PVE 认证失败，请检查用户名/密码/Realm")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unreachable:PVE 登录返回 HTTP %d", resp.StatusCode)
	}

	var out struct {
		Data struct {
			Ticket             string `json:"ticket"`
			CSRFPreventionToken string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("parse_error:读取登录响应失败: %w", err)
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("parse_error:解析登录响应失败: %w", err)
	}
	if out.Data.Ticket == "" {
		return fmt.Errorf("auth_failed:PVE 未返回 ticket")
	}

	c.mu.Lock()
	c.ticket = out.Data.Ticket
	c.csrf = out.Data.CSRFPreventionToken
	c.mu.Unlock()
	return nil
}

// Logout 退出登录（PVE API 无显式 logout 端点需求，保留以对齐调用方）。
func (c *PVEClient) Logout() {}

// do 执行带认证的 API 请求并解码 data 字段。
func (c *PVEClient) do(ctx context.Context, method, path string, body url.Values, out interface{}) error {
	var reqBody io.Reader
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/api2/json"+path, reqBody)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	c.mu.Lock()
	ticket := c.ticket
	csrf := c.csrf
	c.mu.Unlock()
	req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: ticket})

	switch {
	case body != nil && (method == http.MethodPost || method == http.MethodPut):
		encoded := body.Encode()
		req.Body = io.NopCloser(strings.NewReader(encoded))
		req.ContentLength = int64(len(encoded))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if csrf != "" {
			req.Header.Set("CSRFPreventionToken", csrf)
		}
	case body != nil:
		raw, _ := json.Marshal(body)
		req.Body = io.NopCloser(bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("unreachable:请求 PVE 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("parse_error:读取响应失败: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("auth_failed:PVE ticket 失效或权限不足（HTTP %d）", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PVE API 返回 HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("parse_error:解析响应失败: %w", err)
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return fmt.Errorf("parse_error:解析 data 字段失败: %w", err)
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Version 获取 PVE 版本。
func (c *PVEClient) Version() (*PVEVersionResponse, error) {
	var v PVEVersionResponse
	if err := c.do(context.Background(), http.MethodGet, "/version", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Nodes 列出集群节点。
func (c *PVEClient) Nodes() ([]PVENode, error) {
	var nodes []PVENode
	if err := c.do(context.Background(), http.MethodGet, "/nodes", nil, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// NodeStatus 获取节点状态。
func (c *PVEClient) NodeStatus(node string) (*PVENodeStatus, error) {
	var st PVENodeStatus
	if err := c.do(context.Background(), http.MethodGet, "/nodes/"+url.PathEscape(node)+"/status", nil, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// ListGuests 遍历所有在线节点，列出 QEMU 虚拟机 + LXC 容器。
func (c *PVEClient) ListGuests(ctx context.Context) ([]PVEGuest, error) {
	nodes, err := c.Nodes()
	if err != nil {
		return nil, err
	}

	guests := make([]PVEGuest, 0, 32)
	for _, node := range nodes {
		if node.Status != "online" {
			continue
		}

		var qemu []struct {
			PVEGuest
			Template int `json:"template"`
		}
		if err := c.do(ctx, http.MethodGet, "/nodes/"+url.PathEscape(node.Node)+"/qemu", nil, &qemu); err == nil {
			for _, vm := range qemu {
				g := vm.PVEGuest
				g.Node = node.Node
				g.GuestType = "qemu"
				g.Template = vm.Template == 1
				guests = append(guests, g)
			}
		}

		var lxc []PVEGuest
		if err := c.do(ctx, http.MethodGet, "/nodes/"+url.PathEscape(node.Node)+"/lxc", nil, &lxc); err == nil {
			for _, ct := range lxc {
				ct.Node = node.Node
				ct.GuestType = "lxc"
				guests = append(guests, ct)
			}
		}
	}
	return guests, nil
}

// PowerGuest 对虚拟机/容器执行电源操作。
// action 取值：start / stop / reboot / shutdown / reset / suspend / resume。
func (c *PVEClient) PowerGuest(ctx context.Context, node, guestType string, vmid int, action string) error {
	guestType = strings.ToLower(guestType)
	if guestType != "qemu" && guestType != "lxc" {
		return fmt.Errorf("无效的 guest 类型: %s", guestType)
	}
	if !map[string]bool{"start": true, "stop": true, "reboot": true, "shutdown": true, "reset": true, "suspend": true, "resume": true}[action] {
		return fmt.Errorf("无效的电源操作: %s", action)
	}
	// LXC 支持 start/stop/shutdown/reboot；qemu 支持 start/stop/reset/suspend/resume
	if guestType == "lxc" && (action == "reset" || action == "suspend" || action == "resume") {
		return fmt.Errorf("LXC 容器不支持 %s 操作", action)
	}

	path := fmt.Sprintf("/nodes/%s/%s/%d/status/%s",
		url.PathEscape(node), guestType, vmid, url.PathEscape(action))
	return c.do(ctx, http.MethodPost, path, url.Values{}, nil)
}

// ListStorage 遍历所有在线节点，列出数据存储。
func (c *PVEClient) ListStorage(ctx context.Context) ([]PVEStorage, error) {
	nodes, err := c.Nodes()
	if err != nil {
		return nil, err
	}

	storages := make([]PVEStorage, 0, 16)
	for _, node := range nodes {
		if node.Status != "online" {
			continue
		}
		var list []PVEStorage
		if err := c.do(ctx, http.MethodGet, "/nodes/"+url.PathEscape(node.Node)+"/storage", nil, &list); err != nil {
			continue
		}
		for _, s := range list {
			s.Node = node.Node
			storages = append(storages, s)
		}
	}
	return storages, nil
}
