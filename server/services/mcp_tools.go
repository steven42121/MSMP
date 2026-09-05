package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"MSMP/server/db"
	"MSMP/server/models"
)

// ToolDefinition 描述一个可用的工具
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  string `json:"parameters"` // JSON schema string
	Safe        bool   `json:"safe"`     // true = 只读操作直接执行，false = 需要审批
}

// ToolResult 工具执行结果
type ToolResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

var toolDefs = []ToolDefinition{
	{
		Name:        "list_hosts",
		Description: "列出当前租户的所有主机及其状态（在线/离线/待接入）",
		Parameters:  `{"type":"object","properties":{}}`,
		Safe:        true,
	},
	{
		Name:        "get_host_status",
		Description: "获取指定主机的详细状态信息（CPU/内存/磁盘/负载/最近告警）",
		Parameters:  `{"type":"object","properties":{"hostname":{"type":"string","description":"主机名或IP"}}}`,
		Safe:        true,
	},
	{
		Name:        "get_recent_alerts",
		Description: "获取最近 N 条告警列表",
		Parameters:  `{"type":"object","properties":{"limit":{"type":"integer","description":"数量，默认10，最大100"}}}`,
		Safe:        true,
	},
	{
		Name:        "execute_command",
		Description: "在指定主机上执行 shell 命令（需要用户审批）",
		Parameters:  `{"type":"object","properties":{"hostname":{"type":"string","description":"目标主机名或IP"},"command":{"type":"string","description":"要执行的命令"}}}`,
		Safe:        false,
	},
	{
		Name:        "check_service",
		Description: "检查指定服务的运行状态（只读操作）",
		Parameters:  `{"type":"object","properties":{"hostname":{"type":"string","description":"目标主机名或IP"},"service":{"type":"string","description":"服务名称，如 nginx, docker"}}}`,
		Safe:        true,
	},
	{
		Name:        "view_logs",
		Description: "查看指定主机上的日志文件末尾内容（只读操作）",
		Parameters:  `{"type":"object","properties":{"hostname":{"type":"string","description":"目标主机名或IP"},"log_path":{"type":"string","description":"日志文件路径"},"lines":{"type":"integer","description":"读取末尾行数，默认50，最大500"}}}`,
		Safe:        true,
	},
	{
		Name:        "generate_report",
		Description: "生成系统健康报告",
		Parameters:  `{"type":"object","properties":{}}`,
		Safe:        true,
	},
	{
		Name:        "flush_caches",
		Description: "清理Linux内存缓存（需要root权限，会短暂影响性能）",
		Parameters:  `{"type":"object","properties":{"hostname":{"type":"string","description":"目标主机名或IP"},"cache_type":{"type":"string","description":"清理类型：pages(页缓存), dentries(目录缓存), inodes(索引节点), all(全部)"}}}`,
		Safe:        false,
	},
}

// GetToolDefinitions 返回所有可用工具定义
func GetToolDefinitions() []ToolDefinition {
	return toolDefs
}

// ValidateToolCall 验证工具调用是否合法
// 返回值语义：
//   - nil → 工具可直接执行（Safe=true 且参数合法）
//   - "requires_approval" → 需要人工审批（Safe=false 且参数合法）
//   - error → 调用被拒绝（参数非法或包含危险操作）
func ValidateToolCall(name string, args map[string]interface{}) error {
	var found *ToolDefinition
	for i := range toolDefs {
		if toolDefs[i].Name == name {
			found = &toolDefs[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("未知工具: %s", name)
	}

	// 安全检查：危险命令检测（对所有工具生效，不因 Safe 标记而跳过）
	if err := checkDangerousCommand(name, args); err != nil {
		return err
	}

	// 参数合法性校验（针对有参数的工具）
	if err := validateToolParams(name, args); err != nil {
		return err
	}

	// Safe=false 的工具必须走审批流程
	if !found.Safe {
		return fmt.Errorf("requires_approval")
	}

	return nil
}

// checkDangerousCommand 检测危险命令，对所有工具生效
func checkDangerousCommand(name string, args map[string]interface{}) error {
	// flush_caches 工具允许执行特定的缓存清理命令
	if name == "flush_caches" {
		cacheType, _ := args["cache_type"].(string)
		if cacheType != "" && cacheType != "pages" && cacheType != "dentries" && cacheType != "inodes" && cacheType != "all" {
			return fmt.Errorf("无效的 cache_type: %s (允许: pages, dentries, inodes, all)", cacheType)
		}
		return nil
	}

	if name != "execute_command" {
		return nil
	}
	cmd, ok := args["command"].(string)
	if !ok || cmd == "" {
		return fmt.Errorf("execute_command 需要非空的 command 参数")
	}
	// 规范化：去除多余空格，统一为小写，去除控制字符
	normalized := normalizeCommand(cmd)
	// 精确匹配整个命令片段，防止通过路径拼接绕过
	banned := []string{
		"rm -rf", "rm -r ", "rm -fr", "rm -rf /",
		"mkfs", "dd if=", "dd of=",
		":(){:|:&};:", ":(){:|:&}",
	}
	for _, b := range banned {
		if strings.Contains(normalized, b) {
			return fmt.Errorf("命令包含危险操作（%s），已拒绝", b)
		}
	}
	// 检测管道+shell 注入
	if strings.Contains(normalized, "| sh") || strings.Contains(normalized, "|bash") ||
		strings.Contains(normalized, " |sh") || strings.Contains(normalized, " |bash") ||
		strings.Contains(normalized, "; rm ") || strings.Contains(normalized, "; dd ") ||
		strings.Contains(normalized, "&& rm ") || strings.Contains(normalized, "&& dd ") {
		return fmt.Errorf("命令包含管道/序列注入风险，已拒绝")
	}
	return nil
}

// normalizeCommand 规范化命令用于安全检测：转小写、压缩空格、去控制字符
func normalizeCommand(cmd string) string {
	var sb strings.Builder
	sb.Grow(len(cmd))
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		if c >= 32 && c < 127 {
			sb.WriteByte(c)
		}
	}
	s := strings.ToLower(sb.String())
	// 压缩连续空格
	var out strings.Builder
	out.Grow(len(s))
	prevSpace := false
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if !prevSpace {
				out.WriteByte(' ')
			}
			prevSpace = true
		} else {
			out.WriteByte(s[i])
			prevSpace = false
		}
	}
	return out.String()
}

// validateToolParams 校验各工具的参数合法性
func validateToolParams(name string, args map[string]interface{}) error {
	switch name {
	case "get_host_status":
		if _, ok := args["hostname"]; !ok || args["hostname"].(string) == "" {
			return fmt.Errorf("get_host_status 需要 hostname 参数")
		}
	case "get_recent_alerts":
		if l, ok := args["limit"].(float64); ok {
			n := int(l)
			if n < 1 || n > 100 {
				return fmt.Errorf("limit 必须在 1-100 之间，当前值: %d", n)
			}
		}
	case "check_service":
		svc, ok := args["service"].(string)
		if !ok || svc == "" {
			return fmt.Errorf("check_service 需要 service 参数")
		}
		if !validServiceName(svc) {
			return fmt.Errorf("service 参数非法，只允许字母、数字、横线和下划线，当前值: %s", svc)
		}
	case "view_logs":
		logPath, ok := args["log_path"].(string)
		if !ok || logPath == "" {
			return fmt.Errorf("view_logs 需要 log_path 参数")
		}
		if !validLogPath(logPath) {
			return fmt.Errorf("log_path 非法，不允许 ../ 或绝对路径以外的路径")
		}
		if l, ok := args["lines"].(float64); ok {
			n := int(l)
			if n < 1 || n > 500 {
				return fmt.Errorf("lines 必须在 1-500 之间，当前值: %d", n)
			}
		}
	}
	return nil
}

// validServiceName 校验服务名称只包含安全字符
var servicePattern = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

func validServiceName(s string) bool {
	return servicePattern.MatchString(s) && utf8.RuneCountInString(s) <= 64
}

// validLogPath 校验日志路径不含有穿越或危险字符
var logPathPattern = regexp.MustCompile(`^[/][a-zA-Z0-9_\-/.]+$`)

func validLogPath(p string) bool {
	// 不允许 .. 穿越
	if strings.Contains(p, "..") {
		return false
	}
	// 必须是绝对路径且只含安全字符
	if !strings.HasPrefix(p, "/") {
		return false
	}
	return logPathPattern.MatchString(p) && utf8.RuneCountInString(p) <= 256
}

// FindHostByIDOrName 根据 ID（纯数字）或主机名/IP 查找主机
func FindHostByIDOrName(tenantID uint, identifier string) (*models.Host, error) {
	var host models.Host
	// 纯数字 → 按 ID 查询
	if _, err := strconv.ParseUint(identifier, 10, 32); err == nil {
		if err := db.DB.Where("id = ? AND tenant_id = ?", identifier, tenantID).First(&host).Error; err == nil {
			return &host, nil
		}
	}
	// 按 hostname 或 ip 查询
	if err := db.DB.Where("tenant_id = ? AND (hostname = ? OR ip = ?)", tenantID, identifier, identifier).First(&host).Error; err != nil {
		return nil, fmt.Errorf("找不到主机: %s", identifier)
	}
	return &host, nil
}

// ExecuteTool 执行工具调用（仅限安全/只读操作）
func ExecuteTool(name string, args map[string]interface{}, tenantID uint) (*ToolResult, error) {
	switch name {
	case "list_hosts":
		return toolListHosts(tenantID)
	case "get_host_status":
		return toolGetHostStatus(args, tenantID)
	case "get_recent_alerts":
		return toolGetRecentAlerts(args, tenantID)
	case "generate_report":
		return toolGenerateReport(tenantID)
	default:
		return nil, fmt.Errorf("工具 %s 需要审批，请通过审批流程执行", name)
	}
}

func toolListHosts(tenantID uint) (*ToolResult, error) {
	var hosts []models.Host
	db.DB.Where("tenant_id = ?", tenantID).Find(&hosts)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("共 %d 台主机：\n", len(hosts)))
	for _, h := range hosts {
		sb.WriteString(fmt.Sprintf("- [%d] %s (%s) 状态:%s\n", h.ID, h.Hostname, h.IP, h.Status))
	}
	return &ToolResult{Success: true, Output: sb.String()}, nil
}

func toolGetHostStatus(args map[string]interface{}, tenantID uint) (*ToolResult, error) {
	hostname, _ := args["hostname"].(string)
	if hostname == "" {
		return nil, fmt.Errorf("缺少 hostname 参数")
	}

	host, err := FindHostByIDOrName(tenantID, hostname)
	if err != nil {
		return &ToolResult{Success: false, Output: err.Error()}, nil
	}

	var latestMetric models.MetricSample
	db.DB.Where("host_id = ?", host.ID).
		Order("timestamp DESC").
		First(&latestMetric)

	diskPct := 0.0
	if latestMetric.DiskTotal > 0 {
		diskPct = float64(latestMetric.DiskUsed) / float64(latestMetric.DiskTotal) * 100
	}

	var alertCount int64
	db.DB.Model(&models.HostEvent{}).
		Where("host_id = ? AND type = 'alert' AND created_at > ?", host.ID, time.Now().Add(-24*time.Hour)).
		Count(&alertCount)

	output := fmt.Sprintf(`主机: %s (%s)
  操作系统: %s
  状态: %s
  CPU: %.1f%%  内存: %.1f%%  磁盘: %.1f%%  负载: %.2f
  最近指标时间: %s
  近 24h 告警数: %d`,
		host.Hostname, host.IP, host.OS, host.Status,
		latestMetric.CPUPercent, latestMetric.MemPercent, diskPct, latestMetric.Load1,
		latestMetric.Timestamp.Format("2006-01-02 15:04:05"), alertCount,
	)

	return &ToolResult{Success: true, Output: output}, nil
}

func toolGetRecentAlerts(args map[string]interface{}, tenantID uint) (*ToolResult, error) {
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit < 1 {
			limit = 1
		} else if limit > 100 {
			limit = 100
		}
	}

	var alerts []models.HostEvent
	db.DB.Where("tenant_id = ? AND type = 'alert'", tenantID).
		Order("created_at DESC").
		Limit(limit).
		Find(&alerts)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("最近 %d 条告警：\n", len(alerts)))
	for _, a := range alerts {
		sb.WriteString(fmt.Sprintf("[%s] %d - %s (%s)\n", a.Level, a.HostID, a.Message, a.CreatedAt.Format("01-02 15:04")))
	}
	return &ToolResult{Success: true, Output: sb.String()}, nil
}

func toolGenerateReport(tenantID uint) (*ToolResult, error) {
	report, err := GenerateReport(tenantID)
	if err != nil {
		return &ToolResult{Success: false, Output: err.Error()}, nil
	}
	return &ToolResult{Success: true, Output: report}, nil
}

