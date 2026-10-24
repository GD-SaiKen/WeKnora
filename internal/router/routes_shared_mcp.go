package router

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/mcpserver"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

const sharedMCPPath = "/mcp"

// RegisterSharedMCPRoutes mounts OAuth discovery, token endpoints, and the fixed /mcp surface.
func RegisterSharedMCPRoutes(
	r *gin.Engine,
	oauth *handler.SharedMCPOAuthHandler,
	srv *mcpserver.Server,
	sharedMCP interfaces.SharedMCPService,
	tenantSvc interfaces.TenantService,
) {
	if oauth == nil || srv == nil || sharedMCP == nil {
		return
	}
	r.GET("/.well-known/oauth-protected-resource", oauth.ProtectedResourceMetadata)
	r.GET("/.well-known/oauth-authorization-server", oauth.AuthorizationServerMetadata)

	auth := middleware.SharedMCPAuth(sharedMCP, tenantSvc)
	h := gin.WrapH(srv.Handler())
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodDelete} {
		r.Handle(method, sharedMCPPath, auth, h)
	}
}

// RegisterSharedMCPManagementRoutes registers authenticated management APIs.
func RegisterSharedMCPManagementRoutes(
	r *gin.RouterGroup,
	oauth *handler.SharedMCPOAuthHandler,
	cfg *handler.SharedMCPConfigHandler,
	g *rbacGuards,
) {
	if oauth == nil || cfg == nil {
		return
	}
	oauthGrp := r.Group("/oauth/mcp")
	oauthGrp.POST("/register", oauth.RegisterClient)
	oauthGrp.POST("/token", oauth.Token)

	authGrp := g.apiKeyGroup(r.Group("/oauth/mcp"), apiKeyFullAccess())
	authGrp.POST("/authorize", g.Viewer(), oauth.AuthorizeConfirm)
	authGrp.GET("/authorize/tenants", g.Viewer(), oauth.AuthorizeTenants)
	authGrp.GET("/grants", g.Viewer(), oauth.ListGrants)
	authGrp.DELETE("/grants/:grant_id", g.Viewer(), oauth.RevokeGrant)

	cfgGrp := g.apiKeyGroup(r.Group("/shared-mcp"), apiKeyFullAccess())
	cfgGrp.GET("", g.Viewer(), cfg.GetConfig)
	cfgGrp.PUT("", g.Admin(), cfg.UpdateConfig)
}
