package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
)

// MCPApproval 工具调用审批记录
type MCPApproval struct {
	ID         uint       `json:"id"`
	TenantID   uint       `json:"tenant_id"`
	UserID     uint       `json:"user_id"`
	ToolName   string     `json:"tool_name"`
	Arguments  string     `json:"arguments"` // JSON string
	Message    string     `json:"message"`   // 给用户的操作描述
	Status     string     `json:"status"`    // pending/approved/rejected/failed
	Result     string     `json:"result"`    // 执行结果
	CreatedAt  time.Time  `json:"created_at"`
	ExecutedAt *time.Time `json:"executed_at,omitempty"`
}

var (
	approvals      map[uint]*MCPApproval
	approvalsMu    sync.RWMutex
	nextApprovalID uint
)

func init() {
	approvals = make(map[uint]*MCPApproval)
	// 定期清理已完成审批（每30分钟），防止内存持续增长
	go periodicApprovalCleanup(30 * time.Minute)
}

// periodicApprovalCleanup 定期清理已完成审批，防止内存持续增长
func periodicApprovalCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		RemoveCompletedApprovals(10 * time.Minute)
	}
}

// RemoveCompletedApprovals 移除 N 分钟前完成的审批（approved/rejected/failed）
func RemoveCompletedApprovals(age time.Duration) {
	threshold := time.Now().Add(-age)
	approvalsMu.Lock()
	defer approvalsMu.Unlock()
	for id, a := range approvals {
		if a.Status != "pending" && a.ExecutedAt != nil && a.ExecutedAt.Before(threshold) {
			delete(approvals, id)
		}
	}
}

// CreateApproval 创建一个新的工具调用审批请求
func CreateApproval(tenantID, userID uint, toolName string, args map[string]interface{}, description string) (*MCPApproval, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("序列化参数失败: %w", err)
	}

	approvalsMu.Lock()
	id := nextApprovalID
	nextApprovalID++
	approval := &MCPApproval{
		ID:        id,
		TenantID:  tenantID,
		UserID:    userID,
		ToolName:  toolName,
		Arguments: string(argsJSON),
		Message:   description,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	approvals[id] = approval
	approvalsMu.Unlock()

	return approval, nil
}

// GetApproval 获取审批请求
func GetApproval(id uint) (*MCPApproval, bool) {
	approvalsMu.RLock()
	defer approvalsMu.RUnlock()
	a, ok := approvals[id]
	if !ok {
		return nil, false
	}
	// 返回副本，防止调用方修改内部状态
	copy := *a
	return &copy, true
}

// Approve 批准工具调用并执行（内部，由 Controller 调用）
func Approve(id uint, tenantID, userID uint, isAdmin bool) (*MCPApproval, error) {
	approvalsMu.Lock()
	defer approvalsMu.Unlock()

	a, ok := approvals[id]
	if !ok || a.Status != "pending" || a.TenantID != tenantID {
		return nil, fmt.Errorf("审批不存在或已处理")
	}
	if !isAdmin {
		return nil, fmt.Errorf("无权限：仅管理员可批准工具调用")
	}

	a.Status = "approved"
	a.UserID = userID

	// 异步执行
	go executeApproval(a)
	return a, nil
}

// Reject 拒绝工具调用（内部，由 Controller 调用）
func Reject(id uint, tenantID uint, isAdmin bool) (*MCPApproval, error) {
	approvalsMu.Lock()
	defer approvalsMu.Unlock()

	a, ok := approvals[id]
	if !ok || a.Status != "pending" || a.TenantID != tenantID {
		return nil, fmt.Errorf("审批不存在或已处理")
	}
	if !isAdmin {
		return nil, fmt.Errorf("无权限：仅管理员可拒绝工具调用")
	}

	a.Status = "rejected"
	return a, nil
}

// GetPendingApprovals 获取当前租户的待审批 ID 列表（线程安全，返回副本）
func GetPendingApprovals(tenantID uint) []MCPApproval {
	approvalsMu.RLock()
	defer approvalsMu.RUnlock()

	var result []MCPApproval
	for _, a := range approvals {
		if a.TenantID == tenantID && a.Status == "pending" {
			result = append(result, *a) // 值拷贝，安全返回
		}
	}
	return result
}

func executeApproval(a *MCPApproval) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MCP] 执行 panic: %v", r)
		}
		now := time.Now()
		a.ExecutedAt = &now
		// 写回内存
		approvalsMu.Lock()
		if existing, ok := approvals[a.ID]; ok {
			existing.Status = a.Status
			existing.Result = a.Result
			existing.ExecutedAt = a.ExecutedAt
		}
		approvalsMu.Unlock()
	}()

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(a.Arguments), &args); err != nil {
		a.Status = "failed"
		a.Result = "参数解析失败: " + err.Error()
		return
	}

	switch a.ToolName {
	case "execute_command":
		a.Result = execCommand(a, args)
	case "check_service":
		a.Result = checkService(a, args)
	case "view_logs":
		a.Result = viewLogs(a, args)
	case "flush_caches":
		a.Result = flushCaches(a, args)
	default:
		// 安全工具直接执行
		result, err := ExecuteTool(a.ToolName, args, a.TenantID)
		if err != nil {
			a.Status = "failed"
			a.Result = err.Error()
		} else {
			a.Result = result.Output
		}
	}
}

func execCommand(a *MCPApproval, args map[string]interface{}) string {
	hostname, _ := args["hostname"].(string)
	command, _ := args["command"].(string)

	if hostname == "" || command == "" {
		return "参数不完整：需要 hostname 和 command"
	}

	// 安全检查：危险命令检测
	if err := checkDangerousCommand(a.ToolName, args); err != nil {
		return err.Error()
	}

	host, err := FindHostByIDOrName(a.TenantID, hostname)
	if err != nil {
		return fmt.Sprintf("找不到主机 %s: %v", hostname, err)
	}

	// 通过任务系统提交命令
	task := models.Task{
		TenantID:   a.TenantID,
		HostID:     host.ID,
		Type:       "shell",
		Command:    command,
		Status:     "pending",
		CreatedBy:  a.UserID,
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return fmt.Sprintf("提交任务失败: %v", err)
	}

	return fmt.Sprintf("命令已提交至主机 %s (%s)，任务ID=%d。Agent 将执行后返回结果。",
		host.Hostname, host.IP, task.ID)
}

func checkService(a *MCPApproval, args map[string]interface{}) string {
	hostname, _ := args["hostname"].(string)
	service, _ := args["service"].(string)

	if hostname == "" || service == "" {
		return "参数不完整：需要 hostname 和 service"
	}

	host, err := FindHostByIDOrName(a.TenantID, hostname)
	if err != nil {
		return fmt.Sprintf("找不到主机 %s: %v", hostname, err)
	}

	// service 已在 ValidateToolCall 中通过 validServiceName 净化，此处直接使用
	cmd := fmt.Sprintf("systemctl status %s 2>/dev/null || service %s status 2>/dev/null || echo 'service not found'", service, service)
	task := models.Task{
		TenantID: a.TenantID,
		HostID:   host.ID,
		Type:     "shell",
		Command:  cmd,
		Status:   "pending",
		CreatedBy: a.UserID,
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return fmt.Sprintf("提交任务失败: %v", err)
	}

	return fmt.Sprintf("正在检查主机 %s (%s) 上的服务 %s，任务ID=%d", host.Hostname, host.IP, service, task.ID)
}

func viewLogs(a *MCPApproval, args map[string]interface{}) string {
	hostname, _ := args["hostname"].(string)
	logPath, _ := args["log_path"].(string)
	lines := 50
	if l, ok := args["lines"].(float64); ok {
		n := int(l)
		if n < 1 {
			n = 1
		} else if n > 500 {
			n = 500
		}
		lines = n
	}

	if hostname == "" || logPath == "" {
		return "参数不完整：需要 hostname 和 log_path"
	}

	host, err := FindHostByIDOrName(a.TenantID, hostname)
	if err != nil {
		return fmt.Sprintf("找不到主机 %s: %v", hostname, err)
	}

	// logPath 已在 ValidateToolCall 中通过 validLogPath 净化，此处直接使用
	cmd := fmt.Sprintf("tail -n %d %s 2>/dev/null || echo '无法读取日志'", lines, logPath)
	task := models.Task{
		TenantID: a.TenantID,
		HostID:   host.ID,
		Type:     "shell",
		Command:  cmd,
		Status:   "pending",
		CreatedBy: a.UserID,
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return fmt.Sprintf("提交任务失败: %v", err)
	}

	return fmt.Sprintf("正在查看主机 %s (%s) 的日志 %s（最后 %d 行），任务ID=%d",
		host.Hostname, host.IP, logPath, lines, task.ID)
}

// GetToolResult 获取任务执行结果（由 agent 回调）
func GetToolResult(taskID uint) (string, error) {
	var task models.Task
	if err := db.DB.First(&task, taskID).Error; err != nil {
		return "", err
	}
	if task.Result == "" {
		return "任务尚未完成，请稍后重试", nil
	}
	return task.Result, nil
}

// flushCaches 清理Linux内存缓存
func flushCaches(a *MCPApproval, args map[string]interface{}) string {
	hostname, _ := args["hostname"].(string)
	cacheType, _ := args["cache_type"].(string)

	if hostname == "" {
		return "参数不完整：需要 hostname"
	}

	host, err := FindHostByIDOrName(a.TenantID, hostname)
	if err != nil {
		return fmt.Sprintf("找不到主机 %s: %v", hostname, err)
	}

	// 确定要执行的命令
	var cmd string
	switch cacheType {
	case "pages":
		cmd = "echo 1 > /proc/sys/vm/drop_caches"
	case "dentries":
		cmd = "echo 2 > /proc/sys/vm/drop_caches"
	case "inodes":
		cmd = "echo 4 > /proc/sys/vm/drop_caches"
	case "all", "":
		cmd = "echo 3 > /proc/sys/vm/drop_caches"
	default:
		cmd = "echo 3 > /proc/sys/vm/drop_caches"
	}

	task := models.Task{
		TenantID: a.TenantID,
		HostID:   host.ID,
		Type:     "shell",
		Command:  cmd,
		Status:   "pending",
		CreatedBy: a.UserID,
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return fmt.Sprintf("提交任务失败: %v", err)
	}

	return fmt.Sprintf("已提交清理缓存任务至主机 %s (%s)，类型: %s，任务ID=%d",
		host.Hostname, host.IP, cacheType, task.ID)
}

// RunRawCommand 直接通过 SSH/WinRM 运行命令（占位实现）
func RunRawCommand(ctx context.Context, hostID uint, command string, _ interface{}) (string, error) {
	_ = ctx
	_ = command
	var host models.Host
	if err := db.DB.First(&host, hostID).Error; err != nil {
		return "", fmt.Errorf("主机不存在")
	}
	_ = host
	return "", fmt.Errorf("直接命令执行需通过任务系统，请使用 execute_command 工具并提交审批")
}
