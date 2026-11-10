package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/integration/dshwork"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// DshWorkIntegrationHandler serves dsh-work server-to-server integration APIs.
type DshWorkIntegrationHandler struct {
	userService interfaces.UserService
	resolver    *dshwork.SessionResolver
}

// NewDshWorkIntegrationHandler wires integration endpoints.
func NewDshWorkIntegrationHandler(
	userService interfaces.UserService,
	directoryRepo interfaces.DirectoryIdentityRepository,
) *DshWorkIntegrationHandler {
	return &DshWorkIntegrationHandler{
		userService: userService,
		resolver: &dshwork.SessionResolver{
			Users:     userService,
			Directory: directoryRepo,
		},
	}
}

func (h *DshWorkIntegrationHandler) authorizeIntegration(c *gin.Context) bool {
	if !dshwork.Configured() {
		dshwork.WriteNotConfigured(c)
		return false
	}
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if err := dshwork.ValidateBearer(authHeader); err != nil {
		if err == dshwork.ErrUnauthorizedSecret {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return false
		}
		dshwork.WriteNotConfigured(c)
		return false
	}
	return true
}

type dshWorkSessionRequest struct {
	Subject     string `json:"subject"`
	AIHubUserID string `json:"ai_hub_user_id"`
	UserID      string `json:"user_id"`
}

// CreateSession exchanges directory or bound user identity for an access token.
func (h *DshWorkIntegrationHandler) CreateSession(c *gin.Context) {
	if !h.authorizeIntegration(c) {
		return
	}
	var req dshWorkSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	user, err := h.resolver.ResolveUser(c.Request.Context(), dshwork.SessionExchangeRequest{
		Subject:     req.Subject,
		AIHubUserID: req.AIHubUserID,
		UserID:      req.UserID,
	})
	if err != nil {
		switch err {
		case dshwork.ErrAccountDisabled:
			c.JSON(http.StatusForbidden, gin.H{"error": "account_disabled"})
		case dshwork.ErrInvalidSessionRequest:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": "account_not_provisioned"})
		}
		return
	}
	token, tenantID, expiresIn, err := h.userService.IssueDshWorkIntegrationAccessToken(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_issue_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":      user.ID,
		"tenant_id":    tenantID,
		"access_token": token,
		"expires_in":   expiresIn,
	})
}

type dshWorkBindStartRequest struct {
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
}

// StartBindCode registers a pending bind and returns the browser login URL.
func (h *DshWorkIntegrationHandler) StartBindCode(c *gin.Context) {
	if !h.authorizeIntegration(c) {
		return
	}
	var req dshWorkBindStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	enterPath, _, err := dshwork.StartBind(req.RedirectURI, req.State)
	if err != nil {
		if err == dshwork.ErrRedirectNotAllowed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "redirect_not_allowed"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "bind_start_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": dshwork.AbsoluteURL(c, enterPath)})
}

type dshWorkBindExchangeRequest struct {
	BindCode string `json:"bind_code"`
}

// ExchangeBindCode validates a one-time bind code for dsh-work persistence.
func (h *DshWorkIntegrationHandler) ExchangeBindCode(c *gin.Context) {
	if !h.authorizeIntegration(c) {
		return
	}
	var req dshWorkBindExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bind_code_invalid"})
		return
	}
	userID, err := dshwork.ExchangeBindCode(req.BindCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bind_code_invalid"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": userID})
}

// EnterBindBrowser sets the bind cookie and redirects to the SPA login page.
func (h *DshWorkIntegrationHandler) EnterBindBrowser(c *gin.Context) {
	if !dshwork.Configured() {
		dshwork.WriteNotConfigured(c)
		return
	}
	token := strings.TrimSpace(c.Query("token"))
	if !dshwork.EnterBindSession(c, token) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bind_session_invalid"})
		return
	}
	c.Redirect(http.StatusFound, "/login")
}
