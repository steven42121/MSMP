package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"MSMP/server/services"
)

// MCPToolsHandler GET /api/mcp/tools
func MCPToolsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	tools := services.GetToolDefinitions()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	})
}

// MCPProposeHandler POST /api/mcp/propose
// AI 提交工具调用请求：Safe 工具直接执行，非 Safe 工具创建审批记录
func MCPProposeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		ToolName string                 `json:"tool_name"`
		Args     map[string]interface{} `json:"arguments"`
		Message  string                 `json:"message"` // 给用户的操作描述
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.ToolName == "" || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tool_name and message required"})
		return
	}

	tenantID := getTenantID(r)
	userID := getUserID(r)

	// 验证工具调用（含危险命令检测 + 参数净化）
	err := services.ValidateToolCall(req.ToolName, req.Args)
	if err != nil {
		if err.Error() == "requires_approval" {
			// 需要审批：创建审批记录
			approval, createErr := services.CreateApproval(tenantID, userID, req.ToolName, req.Args, req.Message)
			if createErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": createErr.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, map[string]interface{}{
				"action":    "needs_approval",
				"approval":  approval,
				"tool_name": req.ToolName,
				"message":   "此操作需要管理员审批，请等待审批通过后执行",
			})
			return
		}
		// 参数非法或危险命令 → 直接拒绝
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Safe 工具：直接执行，返回结果
	result, execErr := services.ExecuteTool(req.ToolName, req.Args, tenantID)
	if execErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": execErr.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"action":    "executed",
		"tool_name": req.ToolName,
		"result":    result,
	})
}

// MCPApprovalsHandler GET /api/mcp/approvals
func MCPApprovalsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tenantID := getTenantID(r)
	pending := services.GetPendingApprovals(tenantID)

	type shortApproval struct {
		ID        uint   `json:"id"`
		ToolName  string `json:"tool_name"`
		Message   string `json:"message"`
		Status    string `json:"status"`
		UserID    uint   `json:"user_id"`
		CreatedAt string `json:"created_at"`
	}

	items := make([]shortApproval, 0, len(pending))
	for _, a := range pending {
		items = append(items, shortApproval{
			ID:        a.ID,
			ToolName:  a.ToolName,
			Message:   a.Message,
			Status:    a.Status,
			UserID:    a.UserID,
			CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approvals": items,
		"count":     len(items),
	})
}

// MCPApprovalActionHandler POST /api/mcp/approvals/:id/approve 或 /api/mcp/approvals/:id/reject
// 仅管理员可审批
func MCPApprovalActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// 权限校验：仅管理员可审批
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可执行审批操作"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/mcp/approvals/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid approval id"})
		return
	}

	idStr := parts[0]
	action := parts[1]

	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid approval id"})
		return
	}

	tenantID := getTenantID(r)

	switch action {
	case "approve":
		approval, err := services.Approve(id, tenantID, getUserID(r), true)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"approved": true,
			"id":       approval.ID,
			"tool":     approval.ToolName,
			"message":  approval.Message,
		})
	case "reject":
		_, err := services.Reject(id, tenantID, true)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"rejected": "true", "id": fmt.Sprintf("%d", id)})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown action: " + action})
	}
}
