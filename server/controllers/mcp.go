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
// AI 提交工具调用请求，返回待审批记录
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

	if err := services.ValidateToolCall(req.ToolName, req.Args); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	tenantID := getTenantID(r)
	userID := getUserID(r)

	approval, err := services.CreateApproval(tenantID, userID, req.ToolName, req.Args, req.Message)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, approval)
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
		CreatedAt string `json:"created_at"`
	}

	items := make([]shortApproval, 0, len(pending))
	for _, a := range pending {
		items = append(items, shortApproval{
			ID:        a.ID,
			ToolName:  a.ToolName,
			Message:   a.Message,
			Status:    a.Status,
			CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approvals": items,
		"count":     len(items),
	})
}

// MCPApproveHandler POST /api/mcp/approvals/:id/approve
func MCPApproveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/mcp/approvals/")
	idStr = strings.TrimSuffix(idStr, "/approve")

	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid approval id"})
		return
	}

	tenantID := getTenantID(r)
	userID := getUserID(r)

	approval, err := services.Approve(id, tenantID, userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approved": true,
		"id":       approval.ID,
		"message":  approval.Message,
	})
}

// MCPRejectHandler POST /api/mcp/approvals/:id/reject
func MCPRejectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/mcp/approvals/")
	idStr = strings.TrimSuffix(idStr, "/reject")

	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid approval id"})
		return
	}

	tenantID := getTenantID(r)
	_, err := services.Reject(id, tenantID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"rejected": "true", "id": fmt.Sprintf("%d", id)})
}

// MCPApprovalActionHandler POST /api/mcp/approvals/:id/approve 或 /api/mcp/approvals/:id/reject
func MCPApprovalActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
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
		userID := getUserID(r)
		approval, err := services.Approve(id, tenantID, userID)
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
		_, err := services.Reject(id, tenantID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"rejected": "true", "id": fmt.Sprintf("%d", id)})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown action: " + action})
	}
}
