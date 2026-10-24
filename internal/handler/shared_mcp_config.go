package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// SharedMCPConfigHandler manages per-workspace /mcp ceilings.
type SharedMCPConfigHandler struct {
	svc interfaces.SharedMCPService
}

// NewSharedMCPConfigHandler creates the handler.
func NewSharedMCPConfigHandler(svc interfaces.SharedMCPService) *SharedMCPConfigHandler {
	return &SharedMCPConfigHandler{svc: svc}
}

type sharedMCPConfigRequest struct {
	Enabled              *bool     `json:"enabled"`
	Tools                *[]string `json:"tools"`
	KnowledgeBaseIDs     *[]string `json:"knowledge_base_ids"`
	DefaultAgentID       *string   `json:"default_agent_id"`
	RateLimitPerMinute   *int      `json:"rate_limit_per_minute"`
	RedirectURIAllowlist *[]string `json:"redirect_uri_allowlist"`
}

// GetConfig returns the workspace shared MCP configuration.
func (h *SharedMCPConfigHandler) GetConfig(c *gin.Context) {
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	cfg, err := h.svc.GetConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器繁忙，请稍后重试"})
		return
	}
	if cfg == nil {
		cfg = &types.SharedMCPConfig{
			TenantID:           tenantID,
			Enabled:            false,
			Tools:              types.StringArray(types.DefaultSharedMCPTools()),
			KnowledgeBaseIDs:   types.StringArray{},
			RateLimitPerMinute: types.DefaultMCPEndpointRateLimitPerMinute,
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cfg})
}

// UpdateConfig updates the workspace shared MCP configuration.
func (h *SharedMCPConfigHandler) UpdateConfig(c *gin.Context) {
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	var req sharedMCPConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	current, _ := h.svc.GetConfig(c.Request.Context(), tenantID)
	cfg := &types.SharedMCPConfig{TenantID: tenantID}
	if current != nil {
		*cfg = *current
	}
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.Tools != nil {
		cfg.Tools = types.StringArray(*req.Tools)
	}
	if req.KnowledgeBaseIDs != nil {
		cfg.KnowledgeBaseIDs = types.StringArray(*req.KnowledgeBaseIDs)
	}
	if req.DefaultAgentID != nil {
		cfg.DefaultAgentID = *req.DefaultAgentID
	}
	if req.RateLimitPerMinute != nil {
		cfg.RateLimitPerMinute = *req.RateLimitPerMinute
	}
	if req.RedirectURIAllowlist != nil {
		cfg.RedirectURIAllowlist = types.StringArray(*req.RedirectURIAllowlist)
	}
	out, err := h.svc.UpdateConfig(c.Request.Context(), tenantID, cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}
