package router

import (
	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/handler"
)

// RegisterDshWorkIntegrationRoutes registers dsh-work integration endpoints.
func RegisterDshWorkIntegrationRoutes(r *gin.Engine, h *handler.DshWorkIntegrationHandler) {
	if h == nil {
		return
	}
	group := r.Group("/api/v1/integrations/dsh-work")
	{
		group.POST("/sessions", h.CreateSession)
		group.POST("/bind-codes", h.StartBindCode)
		group.POST("/bind-codes/exchange", h.ExchangeBindCode)
		group.GET("/bind/enter", h.EnterBindBrowser)
	}
}
