package mcpserver

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/mark3labs/mcp-go/mcp"
)

func sharedMCPAsEndpoint(call *types.SharedMCPCall) *types.MCPEndpoint {
	return &types.MCPEndpoint{
		TenantID:           call.TenantID,
		KnowledgeBaseIDs:   types.StringArray(call.KnowledgeBaseIDs),
		Tools:              types.StringArray(call.Tools),
		DefaultAgentID:     call.DefaultAgentID,
		Name:               "shared-mcp",
		RateLimitPerMinute: call.RateLimitPerMinute,
	}
}

func sharedToolAllowed(call *types.SharedMCPCall, name string) bool {
	if call == nil {
		return false
	}
	for _, t := range call.Tools {
		if t != name {
			continue
		}
		def, ok := types.LookupMCPEndpointTool(name)
		if ok && def.Destructive && call.Role == types.TenantRoleViewer {
			return false
		}
		return true
	}
	return false
}

func (s *Server) rewriteResourceURLsForShared(
	ctx context.Context,
	call *types.SharedMCPCall,
	result *mcp.CallToolResult,
) *mcp.CallToolResult {
	ep := &types.MCPEndpoint{
		TenantID:           call.TenantID,
		KnowledgeBaseIDs:   types.StringArray(call.KnowledgeBaseIDs),
		Tools:              types.StringArray(call.Tools),
		RateLimitPerMinute: call.RateLimitPerMinute,
	}
	return s.rewriteResourceURLs(ctx, ep, result)
}
