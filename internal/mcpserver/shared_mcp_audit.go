package mcpserver

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/mark3labs/mcp-go/mcp"
)

func sharedMCPToolKBIDs(req mcp.CallToolRequest) []string {
	if ids := req.GetStringSlice("knowledge_base_ids", nil); len(ids) > 0 {
		return ids
	}
	return nil
}

func (s *Server) auditSharedMCPToolCall(
	ctx context.Context,
	call *types.SharedMCPCall,
	tool string,
	allowed bool,
	kbIDs []string,
) {
	if s.auditSvc == nil || call == nil || call.User == nil {
		return
	}
	details := map[string]any{
		"client_id": call.ClientID,
		"tool":      tool,
		"allowed":   allowed,
	}
	if len(kbIDs) > 0 {
		details["knowledge_base_ids"] = kbIDs
	}
	outcome := types.AuditOutcomeSuccess
	if !allowed {
		outcome = types.AuditOutcomeDenied
	}
	raw, _ := json.Marshal(details)
	entry := &types.AuditLog{
		TenantID:    call.TenantID,
		ActorUserID: call.User.ID,
		ActorRole:   string(call.Role),
		Action:      types.AuditActionSharedMCPToolCall,
		ScopeType:   "mcp",
		ScopeID:     call.ClientID,
		TargetType:  "mcp_tool",
		TargetID:    tool,
		Outcome:     outcome,
		Details:     types.JSON(raw),
	}
	go func() {
		bg, cancel := context.WithTimeout(logger.CloneContext(ctx), 5*time.Second)
		defer cancel()
		if err := s.auditSvc.Log(bg, entry); err != nil {
			logger.Warnf(bg, "[mcpserver] shared mcp audit failed: %v", err)
		}
	}()
}

// auditDetailsForTest decodes Details for tests without importing JSON helpers elsewhere.
func auditDetailsForTest(raw types.JSON) map[string]any {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return m
}
