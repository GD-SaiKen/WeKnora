package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/sharedmcp"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// SharedMCPAuth authenticates the fixed /mcp surface with user-issued MCP access tokens.
func SharedMCPAuth(svc interfaces.SharedMCPService, tenantSvc interfaces.TenantService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			origin := sharedmcp.RequestOrigin(c.Request)
			c.Header("WWW-Authenticate", `Bearer realm="weknora-mcp", resource_metadata="`+origin+`/.well-known/oauth-protected-resource"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		if strings.HasPrefix(token, types.MCPEndpointTokenPrefix) {
			c.Header("WWW-Authenticate", `Bearer realm="weknora-mcp"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		call, err := svc.ResolveAccessToken(c.Request.Context(), token, time.Now())
		if err != nil {
			c.Header("WWW-Authenticate", `Bearer realm="weknora-mcp"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		tenant, err := tenantSvc.GetTenantByID(c.Request.Context(), call.TenantID)
		if err != nil || tenant == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		scope := types.MCPEndpointScope(&types.MCPEndpoint{
			TenantID:           call.TenantID,
			KnowledgeBaseIDs: types.StringArray(call.KnowledgeBaseIDs),
			Tools:              types.StringArray(call.Tools),
		})
		applyAuthSession(c, authSession{
			User:        call.User,
			Principal:   types.Principal{Type: types.PrincipalWebUser, ID: call.User.ID},
			TenantID:    call.TenantID,
			Tenant:      tenant,
			Role:        call.Role,
			APIKeyScope: &scope,
			Extra:       map[types.ContextKey]any{types.SharedMCPContextKey: call},
			AuthSource:  types.AuthSourceAPIKey,
		})
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}
