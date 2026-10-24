package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/sharedmcp"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// SharedMCPOAuthHandler serves OAuth metadata and token endpoints for /mcp.
type SharedMCPOAuthHandler struct {
	svc interfaces.SharedMCPService
}

// NewSharedMCPOAuthHandler creates the handler.
func NewSharedMCPOAuthHandler(svc interfaces.SharedMCPService) *SharedMCPOAuthHandler {
	return &SharedMCPOAuthHandler{svc: svc}
}

// ProtectedResourceMetadata serves RFC 9728 metadata.
func (h *SharedMCPOAuthHandler) ProtectedResourceMetadata(c *gin.Context) {
	origin := sharedmcp.RequestOrigin(c.Request)
	c.JSON(http.StatusOK, gin.H{
		"resource":                origin + "/mcp",
		"authorization_servers":   []string{origin},
		"bearer_methods_supported": []string{"header"},
	})
}

// AuthorizationServerMetadata serves RFC 8414 metadata.
func (h *SharedMCPOAuthHandler) AuthorizationServerMetadata(c *gin.Context) {
	origin := sharedmcp.RequestOrigin(c.Request)
	c.JSON(http.StatusOK, gin.H{
		"issuer":                                origin,
		"authorization_endpoint":                origin + "/api/v1/oauth/mcp/authorize",
		"token_endpoint":                        origin + "/api/v1/oauth/mcp/token",
		"registration_endpoint":                 origin + "/api/v1/oauth/mcp/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                   []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	})
}

type sharedMCPRegisterRequest struct {
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

// RegisterClient handles dynamic client registration.
func (h *SharedMCPOAuthHandler) RegisterClient(c *gin.Context) {
	var req sharedMCPRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	client, err := h.svc.RegisterClient(c.Request.Context(), req.ClientName, req.RedirectURIs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_redirect_uri"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"client_id":     client.ID,
		"client_name":   client.Name,
		"redirect_uris": client.RedirectURIs,
	})
}

type sharedMCPAuthorizeRequest struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	TenantID            uint64 `json:"tenant_id"`
}

// AuthorizeConfirm issues an authorization code for the logged-in user.
func (h *SharedMCPOAuthHandler) AuthorizeConfirm(c *gin.Context) {
	userID := c.GetString(types.UserIDContextKey.String())
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	var req sharedMCPAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if strings.TrimSpace(req.CodeChallengeMethod) != "" && req.CodeChallengeMethod != "S256" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "仅支持 S256"})
		return
	}
	code, err := h.svc.IssueAuthorizationCode(
		c.Request.Context(),
		userID,
		req.TenantID,
		req.ClientID,
		req.RedirectURI,
		req.CodeChallenge,
		time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"code":          code,
			"redirect_uri": req.RedirectURI,
		},
	})
}

// AuthorizeTenants lists workspaces the user may bind for /mcp.
func (h *SharedMCPOAuthHandler) AuthorizeTenants(c *gin.Context) {
	userID := c.GetString(types.UserIDContextKey.String())
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	ids, err := h.svc.AuthorizeTenants(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器繁忙，请稍后重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"tenant_ids": ids}})
}

// Token exchanges authorization codes and refresh tokens.
func (h *SharedMCPOAuthHandler) Token(c *gin.Context) {
	grantType := c.PostForm("grant_type")
	now := time.Now()
	switch grantType {
	case "authorization_code":
		code := c.PostForm("code")
		redirectURI := c.PostForm("redirect_uri")
		clientID := c.PostForm("client_id")
		verifier := c.PostForm("code_verifier")
		access, refresh, expiresIn, err := h.svc.ExchangeAuthorizationCode(
			c.Request.Context(), code, redirectURI, clientID, verifier, now,
		)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token":  access,
			"token_type":    "Bearer",
			"expires_in":    expiresIn,
			"refresh_token": refresh,
		})
	case "refresh_token":
		refresh := c.PostForm("refresh_token")
		access, newRefresh, expiresIn, err := h.svc.Refresh(c.Request.Context(), refresh, now)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token":  access,
			"token_type":    "Bearer",
			"expires_in":    expiresIn,
			"refresh_token": newRefresh,
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
	}
}

// RevokeGrant removes a user's MCP authorization.
func (h *SharedMCPOAuthHandler) RevokeGrant(c *gin.Context) {
	userID := c.GetString(types.UserIDContextKey.String())
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	grantID := strings.TrimSpace(c.Param("grant_id"))
	if err := h.svc.RevokeGrant(c.Request.Context(), userID, grantID, time.Now()); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "授权不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": grantID, "revoked": true}})
}

// ListGrants lists the caller's active MCP grants.
func (h *SharedMCPOAuthHandler) ListGrants(c *gin.Context) {
	userID := c.GetString(types.UserIDContextKey.String())
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	grants, err := h.svc.ListGrants(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器繁忙，请稍后重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": grants})
}
