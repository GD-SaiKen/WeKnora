package sharedmcp

import (
	"net/http"
	"strings"
)

// RequestOrigin returns the external origin for OAuth metadata (scheme + host).
func RequestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = strings.ToLower(proto)
	}
	host := strings.TrimSpace(r.Host)
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); fwd != "" {
		host = fwd
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}
