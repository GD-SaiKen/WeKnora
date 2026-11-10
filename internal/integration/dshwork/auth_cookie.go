package dshwork

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BindRedirectAfterLogin reads the bind session cookie and returns a callback URL.
func BindRedirectAfterLogin(c *gin.Context, userID string) (redirectURL string, ok bool) {
	cookie, err := c.Request.Cookie(BindSessionCookieName)
	if err != nil || cookie == nil || strings.TrimSpace(cookie.Value) == "" {
		return "", false
	}
	token := strings.TrimSpace(cookie.Value)
	redirectURL, ok = CompletePendingBind(token, userID)
	if ok {
		clearBindSessionCookie(c)
	}
	return redirectURL, ok
}

// BindRedirectOnLoginFailure sends the browser to dsh-work with bind_error when a bind was pending.
func BindRedirectOnLoginFailure(c *gin.Context) (redirectURL string, ok bool) {
	cookie, err := c.Request.Cookie(BindSessionCookieName)
	if err != nil || cookie == nil {
		return "", false
	}
	token := strings.TrimSpace(cookie.Value)
	redirectURL, ok = FailPendingBind(token)
	if ok {
		clearBindSessionCookie(c)
	}
	return redirectURL, ok
}

// SetBindSessionCookie stores the pending bind session token in the browser.
func SetBindSessionCookie(c *gin.Context, sessionToken string) {
	c.SetCookie(BindSessionCookieName, sessionToken, int(bindTTL.Seconds()), "/", "", false, true)
}

func clearBindSessionCookie(c *gin.Context) {
	c.SetCookie(BindSessionCookieName, "", -1, "/", "", false, true)
}

// EnterBindSession validates the token query param and sets the bind cookie before login.
func EnterBindSession(c *gin.Context, sessionToken string) bool {
	sessionToken = strings.TrimSpace(sessionToken)
	if _, _, ok := LookupPendingBind(sessionToken); !ok {
		return false
	}
	SetBindSessionCookie(c, sessionToken)
	return true
}

// AbsoluteURL builds an absolute URL for the current request host.
func AbsoluteURL(c *gin.Context, path string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if fwd := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); fwd != "" {
		scheme = strings.Split(fwd, ",")[0]
	}
	return scheme + "://" + c.Request.Host + path
}

// WriteNotConfigured responds with 404 when integration is disabled.
func WriteNotConfigured(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
}
