package services

import (
	"fmt"
	"strings"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
)

// ToolDefinition 描述一个可用的工具
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  string `json:"parameters"` // JSON schema string
	Safe        bool   `json:"safe"`     // true = 只读操作，false = 需要审批
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
		Parameters:  `{"type":"object","properties":{"limit":{"type":"integer","description":"数量，默认10"}}}`,
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
		Description: "检查指定服务的运行状态（需要用户审批）",
		Parameters:  `{"type":"object","properties":{"hostname":{"type":"string","description":"目标主机名或IP"},"service":{"type":"string","description":"服务名称，如 nginx, docker"}}}`,
		Safe:        false,
	},
	{
		Name:        "view_logs",
		Description: "查看指定主机上的日志文件（需要用户审批）",
		Parameters:  `{"type":"object","properties":{"hostname":{"type":"string","description":"目标主机名或IP"},"log_path":{"type":"string","description":"日志文件路径"},"lines":{"type":"integer","description":"读取末尾行数，默认50"}}}`,
		Safe:        false,
	},
	{
		Name:        "generate_report",
		Description: "生成系统健康报告",
		Parameters:  `{"type":"object","properties":{}}`,
		Safe:        true,
	},
}

// GetToolDefinitions 返回所有可用工具定义
func GetToolDefinitions() []ToolDefinition {
	return toolDefs
}

// ValidateToolCall 验证工具调用是否合法
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

	if !found.Safe {
		return nil // 非安全操作交给审批流程
	}

	// 安全检查：只读工具不允许执行危险命令
	if name == "execute_command" {
		if cmd, ok := args["command"].(string); ok {
			dangerous := []string{"rm -rf", "dd ", "mkfs", "shutdown", "reboot", "format", ":(){:|:&};:", "亡"}
			for _, d := range dangerous {
				if strings.Contains(cmd, d) {
					return fmt.Errorf("命令包含危险操作，已拒绝: %s", d)
				}
			}
		}
	}
	return nil
}

// FindHostByIDOrName 根据 ID 或主机名/IP 查找主机
func FindHostByIDOrName(tenantID uint, identifier string) (*models.Host, error) {
	var host models.Host
	// 先尝试按 ID 查找
	if _, err := fmt.Sscanf(identifier, "%d", new(uint)); err == nil {
		if err := db.DB.Where("id = ? AND tenant_id = ?", identifier, tenantID).First(&host).Error; err == nil {
			return &host, nil
		}
	}
	// 再尝试按 hostname 或 ip 查找
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
	}

	var alerts []models.HostEvent
	db.DB.Where("tenant_id = ? AND type = 'alert'", tenantID).
		Order("created_at DESC").
		Limit(limit).
		Find(&alerts)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("最近 %d 条告警：\n", len(alerts)))
	for _, a := range alerts {
		sb.WriteString(fmt.Sprintf("[%d] %s - %s (%s)\n", a.Level, a.HostID, a.Message, a.CreatedAt.Format("01-02 15:04")))
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
