package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"MSMP/server/db"
	"MSMP/server/models"
)

// riskPortDef 高风险端口定义。
type riskPortDef struct {
	Service    string
	Level      string // critical | warning
	Suggestion string
}

// riskPorts 内置高风险端口表。
var riskPorts = map[int]riskPortDef{
	22:    {"SSH", "warning", "限制 SSH 仅内网访问，或改用密钥登录并禁用密码认证"},
	23:    {"Telnet", "critical", "Telnet 明文传输，应立即禁用并改用 SSH"},
	3306:  {"MySQL", "critical", "数据库端口勿对公网开放，仅监听 127.0.0.1 或内网"},
	5432:  {"PostgreSQL", "critical", "数据库端口勿对公网开放"},
	6379:  {"Redis", "critical", "Redis 默认无认证，禁止对公网开放"},
	27017: {"MongoDB", "critical", "MongoDB 默认无认证，禁止对公网开放"},
	9200:  {"Elasticsearch", "critical", "Elasticsearch 无认证，禁止对公网开放"},
	9300:  {"Elasticsearch", "warning", "Elasticsearch 节点通信端口勿对外"},
	2375:  {"Docker API", "critical", "Docker Remote API 无 TLS 即可提权，必须关闭"},
	2376:  {"Docker API (TLS)", "warning", "Docker Remote API 需确保仅可信来源访问"},
	11211: {"Memcached", "critical", "Memcached 默认无认证，禁止对公网开放"},
	3389:  {"RDP", "critical", "远程桌面端口勿对公网开放，建议经 VPN"},
	1433:  {"MSSQL", "critical", "数据库端口勿对公网开放"},
	2049:  {"NFS", "warning", "NFS 端口应限制内网访问"},
	445:   {"SMB", "warning", "SMB 端口应限制内网访问"},
	5900:  {"VNC", "warning", "VNC 端口应限制内网访问或使用 SSH 隧道"},
	2379:  {"etcd", "critical", "etcd 存储敏感数据，禁止对公网开放"},
	2380:  {"etcd", "warning", "etcd 节点通信端口勿对外"},
}

// PortRisk 端口暴露风险条目。
type PortRisk struct {
	HostID     uint   `json:"host_id"`
	Hostname   string `json:"hostname"`
	PublicIP   string `json:"public_ip"`
	Port       int    `json:"port"`
	Proto      string `json:"proto"`
	Service    string `json:"service"`
	Level      string `json:"level"`
	Reason     string `json:"reason"`
	Suggestion string `json:"suggestion"`
}

// AuditPortRisks 汇总租户下所有主机的端口暴露风险。
func AuditPortRisks(tenantID uint) []PortRisk {
	var hosts []models.Host
	db.DB.Where("tenant_id = ?", tenantID).Find(&hosts)
	hostMap := make(map[uint]models.Host, len(hosts))
	for _, h := range hosts {
		hostMap[h.ID] = h
	}

	var ports []models.HostPort
	db.DB.Where("tenant_id = ?", tenantID).Order("host_id asc").Find(&ports)

	risks := make([]PortRisk, 0)
	seen := map[string]bool{}
	for _, p := range ports {
		def, ok := riskPorts[p.LocalPort]
		if !ok {
			continue
		}
		h := hostMap[p.HostID]
		key := strconv.Itoa(int(p.HostID)) + ":" + strconv.Itoa(p.LocalPort) + ":" + strings.ToLower(p.Type)
		if seen[key] {
			continue
		}
		seen[key] = true

		level := def.Level
		reason := "监听服务命中高风险端口表"
		if h.PublicIP != "" {
			level = "critical"
			reason = "主机具有公网 IP（" + h.PublicIP + "），高风险端口对外暴露"
		} else if isOpenAddr(p.LocalAddr) {
			if level != "critical" {
				level = "warning"
			}
			reason = "监听 0.0.0.0/::（对外地址），" + reason
		}

		risks = append(risks, PortRisk{
			HostID:     p.HostID,
			Hostname:   h.Hostname,
			PublicIP:   h.PublicIP,
			Port:       p.LocalPort,
			Proto:      p.Type,
			Service:    def.Service,
			Level:      level,
			Reason:     reason,
			Suggestion: def.Suggestion,
		})
	}
	return risks
}

func isOpenAddr(addr string) bool {
	a := strings.TrimSpace(addr)
	return a == "0.0.0.0" || a == "::" || a == "*" || a == ""
}

// runHostCommand 通过 SSH 渠道在主机上执行命令。
func runHostCommand(tenantID, hostID uint, cmd string) (string, error) {
	client, err := dialHostSSH(tenantID, hostID)
	if err != nil {
		return "", err
	}
	defer client.Close()
	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	return strings.TrimSpace(string(out)), err
}

// BaselineItem 基线检查项。
type BaselineItem struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // pass | fail | warn | na
	Detail     string `json:"detail"`
	Suggestion string `json:"suggestion"`
}

// RunBaselineCheck 对主机执行安全基线检查。
func RunBaselineCheck(tenantID, hostID uint) ([]BaselineItem, error) {
	script := `echo "==permit_root=="; grep -i '^PermitRootLogin' /etc/ssh/sshd_config 2>/dev/null || echo "not_found"
echo "==password_auth=="; grep -i '^PasswordAuthentication' /etc/ssh/sshd_config 2>/dev/null || echo "not_found"
echo "==firewall=="; fw="none"; command -v firewall-cmd >/dev/null 2>&1 && fw="firewalld"; command -v ufw >/dev/null 2>&1 && fw="ufw"; echo "$fw"
echo "==firewall_active=="; systemctl is-active firewalld 2>/dev/null; systemctl is-active ufw 2>/dev/null | head -1
echo "==selinux=="; getenforce 2>/dev/null || echo "n/a"
echo "==empty_password=="; awk -F: '($2==""){print $1}' /etc/shadow 2>/dev/null || echo "check_failed"`
	out, err := runHostCommand(tenantID, hostID, script)
	if err != nil {
		return nil, err
	}

	items := []BaselineItem{}
	section := ""
	detail := strings.Builder{}
	flush := func() {
		if section == "" {
			return
		}
		items = append(items, evaluateBaseline(section, strings.TrimSpace(detail.String())))
		section = ""
		detail.Reset()
	}
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "==permit_root=="):
			flush()
			section = "permit_root"
		case strings.HasPrefix(line, "==password_auth=="):
			flush()
			section = "password_auth"
		case strings.HasPrefix(line, "==firewall=="):
			flush()
			section = "firewall"
		case strings.HasPrefix(line, "==firewall_active=="):
			flush()
			section = "firewall_active"
		case strings.HasPrefix(line, "==selinux=="):
			flush()
			section = "selinux"
		case strings.HasPrefix(line, "==empty_password=="):
			flush()
			section = "empty_password"
		default:
			if section != "" {
				detail.WriteString(line)
				detail.WriteString("\n")
			}
		}
	}
	flush()
	return items, nil
}

func evaluateBaseline(section, detail string) BaselineItem {
	base := BaselineItem{Name: sectionName(section), Detail: detail}
	switch section {
	case "permit_root":
		if strings.Contains(strings.ToLower(detail), "no") {
			base.Status, base.Suggestion = "pass", ""
		} else if detail == "not_found" {
			base.Status, base.Suggestion = "warn", "未显式禁用 root 登录，建议配置 PermitRootLogin no"
		} else {
			base.Status, base.Suggestion = "fail", "建议配置 PermitRootLogin no，改用普通用户 + sudo"
		}
	case "password_auth":
		if strings.Contains(strings.ToLower(detail), "no") {
			base.Status, base.Suggestion = "pass", ""
		} else if detail == "not_found" {
			base.Status, base.Suggestion = "warn", "未显式禁用密码登录，建议配置 PasswordAuthentication no"
		} else {
			base.Status, base.Suggestion = "fail", "建议关闭密码登录，改用 SSH 密钥"
		}
	case "firewall":
		if detail == "none" {
			base.Status, base.Suggestion = "fail", "未检测到 firewalld/ufw，建议启用主机防火墙"
		} else {
			base.Status, base.Suggestion = "pass", ""
		}
	case "firewall_active":
		d := strings.ToLower(detail)
		if strings.Contains(d, "running") {
			base.Status, base.Suggestion = "pass", ""
		} else if strings.Contains(d, "inactive") {
			base.Status, base.Suggestion = "fail", "防火墙服务未运行，建议启用"
		} else if strings.Contains(d, "active") {
			base.Status, base.Suggestion = "pass", ""
		} else {
			base.Status, base.Suggestion = "fail", "防火墙服务未运行或未安装，建议启用"
		}
	case "selinux":
		enforced := strings.Contains(strings.ToLower(detail), "enforcing")
		if enforced {
			base.Status, base.Suggestion = "pass", ""
		} else if detail == "n/a" {
			base.Status, base.Suggestion = "na", ""
		} else {
			base.Status, base.Suggestion = "warn", "SELinux 未启用为 Enforcing 模式"
		}
	case "empty_password":
		if detail == "check_failed" {
			base.Status, base.Suggestion = "na", "需要 root 权限检查"
		} else if detail == "" {
			base.Status, base.Suggestion = "pass", ""
		} else {
			base.Status, base.Suggestion = "fail", "存在空密码账户：" + strings.Fields(detail)[0]
		}
	}
	return base
}

func sectionName(section string) string {
	switch section {
	case "permit_root":
		return "SSH 允许 root 登录"
	case "password_auth":
		return "SSH 密码登录"
	case "firewall":
		return "防火墙软件"
	case "firewall_active":
		return "防火墙运行状态"
	case "selinux":
		return "SELinux"
	case "empty_password":
		return "空密码账户"
	default:
		return section
	}
}

// detectFirewall 检测主机防火墙类型。
func detectFirewall(tenantID, hostID uint) (string, error) {
	out, err := runHostCommand(tenantID, hostID,
		`command -v firewall-cmd >/dev/null 2>&1 && echo firewalld; command -v ufw >/dev/null 2>&1 && echo ufw`)
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if strings.Contains(out, "firewalld") {
		return "firewalld", nil
	}
	if strings.Contains(out, "ufw") {
		return "ufw", nil
	}
	return "", nil
}

// GetFirewallRules 返回主机防火墙规则文本。
func GetFirewallRules(tenantID, hostID uint) (string, error) {
	fw, err := detectFirewall(tenantID, hostID)
	if err != nil {
		return "", err
	}
	cmd := ""
	switch fw {
	case "firewalld":
		cmd = "firewall-cmd --list-all 2>/dev/null"
	case "ufw":
		cmd = "ufw status verbose 2>/dev/null"
	default:
		return "", errUnsupportedFirewall()
	}
	out, err := runHostCommand(tenantID, hostID, cmd)
	if err != nil {
		return "", err
	}
	return fw + "\n" + out, nil
}

// ManageFirewall 开关端口。
func ManageFirewall(tenantID, hostID uint, action string, port int, proto string) (string, error) {
	fw, err := detectFirewall(tenantID, hostID)
	if err != nil {
		return "", err
	}
	if proto == "" {
		proto = "tcp"
	}
	svc := strconv.Itoa(port) + "/" + proto
	var cmd string
	switch fw {
	case "firewalld":
		switch action {
		case "open":
			cmd = "firewall-cmd --add-port=" + svc + " --permanent && firewall-cmd --reload"
		case "close":
			cmd = "firewall-cmd --remove-port=" + svc + " --permanent && firewall-cmd --reload"
		default:
			return "", errInvalidAction()
		}
	case "ufw":
		switch action {
		case "open":
			cmd = "ufw allow " + svc
		case "close":
			cmd = "ufw delete allow " + svc
		default:
			return "", errInvalidAction()
		}
	default:
		return "", errUnsupportedFirewall()
	}
	out, err := runHostCommand(tenantID, hostID, cmd)
	if err != nil {
		return "", err
	}
	return fw + " " + action + " " + svc + "\n" + out, nil
}

func errUnsupportedFirewall() error {
	return &securityErr{msg: "无法识别主机防火墙类型（需 firewalld 或 ufw）"}
}
func errInvalidAction() error {
	return &securityErr{msg: "无效操作，仅支持 open/close"}
}

type securityErr struct{ msg string }

func (e *securityErr) Error() string { return e.msg }

// SecurityRisksHandler GET /api/security/risks
func SecurityRisksHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, AuditPortRisks(getTenantID(r)))
}

// SecurityBaselineHandler POST /api/security/baseline/{hostID}
func SecurityBaselineHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	idStr := strings.TrimPrefix(r.URL.Path, "/api/security/baseline/")
	hostID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid host id"})
		return
	}
	items, err := RunBaselineCheck(tenantID, uint(hostID))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "基线检查失败: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// SecurityFirewallHandler GET/POST /api/security/firewall/{hostID}
func SecurityFirewallHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	idStr := strings.TrimPrefix(r.URL.Path, "/api/security/firewall/")
	hostID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid host id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		out, err := GetFirewallRules(tenantID, uint(hostID))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"rules": out})

	case http.MethodPost:
		var req struct {
			Action string `json:"action"`
			Port   int    `json:"port"`
			Proto  string `json:"proto"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if req.Port <= 0 || req.Port > 65535 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid port"})
			return
		}
		out, err := ManageFirewall(tenantID, uint(hostID), req.Action, req.Port, req.Proto)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"result": out})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}