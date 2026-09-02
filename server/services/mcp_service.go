package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"MSMP/server/collectors"
	"MSMP/server/db"
	"MSMP/server/models"
)

// MCPApproval 工具调用审批记录
type MCPApproval struct {
	ID          uint      `json:"id"`
	TenantID    uint      `json:"tenant_id"`
	UserID      uint      `json:"user_id"`
	ToolName    string    `json:"tool_name"`
	Arguments   string    `json:"arguments"` // JSON string
	Message     string    `json:"message"`   // 给用户的操作描述
	Status      string    `json:"status"`    // pending/approved/rejected
	Result      string    `json:"result"`    // 执行结果
	CreatedAt   time.Time `json:"created_at"`
	ExecutedAt  *time.Time `json:"executed_at,omitempty"`
}

var (
	approvals      map[uint]*MCPApproval
	approvalsMu    sync.RWMutex
	nextApprovalID uint = 1000
)

func init() {
	approvals = make(map[uint]*MCPApproval)
}

// CreateApproval 创建一个新的工具调用审批请求
func CreateApproval(tenantID, userID uint, toolName string, args map[string]interface{}, description string) (*MCPApproval, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("序列化参数失败: %w", err)
	}

	approval := &MCPApproval{
		ID:        nextApprovalID,
		TenantID:  tenantID,
		UserID:    userID,
		ToolName:  toolName,
		Arguments: string(argsJSON),
		Message:   description,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	nextApprovalID++

	approvalsMu.Lock()
	approvals[approval.ID] = approval
	approvalsMu.Unlock()

	return approval, nil
}

// GetApproval 获取审批请求
func GetApproval(id uint) (*MCPApproval, bool) {
	approvalsMu.RLock()
	defer approvalsMu.RUnlock()
	a, ok := approvals[id]
	return a, ok
}

// Approve 批准工具调用并执行
func Approve(id uint, tenantID, userID uint) (*MCPApproval, error) {
	approvalsMu.Lock()
	defer approvalsMu.Unlock()

	a, ok := approvals[id]
	if !ok || a.Status != "pending" || a.TenantID != tenantID {
		return nil, fmt.Errorf("审批不存在或已处理")
	}

	a.Status = "approved"
	a.UserID = userID

	// 异步执行
	go executeApproval(a)
	return a, nil
}

// Reject 拒绝工具调用
func Reject(id uint, tenantID uint) (*MCPApproval, error) {
	approvalsMu.Lock()
	defer approvalsMu.Unlock()

	a, ok := approvals[id]
	if !ok || a.Status != "pending" || a.TenantID != tenantID {
		return nil, fmt.Errorf("审批不存在或已处理")
	}

	a.Status = "rejected"
	return a, nil
}

// GetPendingApprovals 获取当前租户的待审批列表
func GetPendingApprovals(tenantID uint) []*MCPApproval {
	approvalsMu.RLock()
	defer approvalsMu.RUnlock()

	var result []*MCPApproval
	for _, a := range approvals {
		if a.TenantID == tenantID && a.Status == "pending" {
			result = append(result, a)
		}
	}
	return result
}

func executeApproval(a *MCPApproval) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MCP] 执行 panic: %v", r)
			a.Status = "failed"
			a.Result = fmt.Sprintf("执行异常: %v", r)
		}
		now := time.Now()
		a.ExecutedAt = &now
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
		lines = int(l)
	}

	if hostname == "" || logPath == "" {
		return "参数不完整：需要 hostname 和 log_path"
	}

	host, err := FindHostByIDOrName(a.TenantID, hostname)
	if err != nil {
		return fmt.Sprintf("找不到主机 %s: %v", hostname, err)
	}

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

// RunRawCommand 直接通过 SSH/WinRM 运行命令（需要 cred service）
func RunRawCommand(ctx context.Context, hostID uint, command string, credSvc CredentialService) (string, error) {
	var host models.Host
	if err := db.DB.First(&host, hostID).Error; err != nil {
		return "", fmt.Errorf("主机不存在")
	}

	var binding models.ChannelBinding
	if err := db.DB.Where("host_id = ?", hostID).First(&binding).Error; err != nil {
		return "", fmt.Errorf("主机未配置采集渠道")
	}

	ch, ok := collectors.NewRegistry().Get(binding.Type)
	if !ok {
		return "", fmt.Errorf("不支持的渠道类型: %s", binding.Type)
	}

	cred := func() (string, error) {
		return credSvc.Decrypt(binding.Credential)
	}

	// 使用 channel 的 run 方法（通过 Collect 接口间接执行）
	// 注意：这里需要根据具体 collector 实现来扩展
	_ = ctx
	_ = ch
	_ = cred
	return "", fmt.Errorf("直接命令执行需要通过任务系统由 Agent 执行")
}
