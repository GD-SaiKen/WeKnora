package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/mcpserver"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type stubSharedMCPService struct {
	interfaces.SharedMCPService
}

func (stubSharedMCPService) ResolveAccessToken(context.Context, string, time.Time) (*types.SharedMCPCall, error) {
	return nil, errors.New("invalid token")
}

func TestSharedMCPUnauthorizedAdvertisesMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	oauth := handler.NewSharedMCPOAuthHandler(stubSharedMCPService{})
	srv := mcpserver.NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	RegisterSharedMCPRoutes(r, oauth, srv, stubSharedMCPService{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Host = "kb.example"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
	got := rec.Header().Get("WWW-Authenticate")
	want := `resource_metadata="https://kb.example/.well-known/oauth-protected-resource"`
	if !strings.Contains(got, want) {
		t.Fatalf("header %s", got)
	}
}

func TestSharedMCPWellKnownMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	oauth := handler.NewSharedMCPOAuthHandler(stubSharedMCPService{})
	srv := mcpserver.NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	RegisterSharedMCPRoutes(r, oauth, srv, stubSharedMCPService{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	req.Host = "kb.example"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("protected resource status %d", rec.Code)
	}
	var pr map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &pr); err != nil {
		t.Fatal(err)
	}
	servers, _ := pr["authorization_servers"].([]any)
	if len(servers) != 1 || servers[0] != "https://kb.example" {
		t.Fatalf("authorization_servers %v", pr["authorization_servers"])
	}
	if pr["resource"] != "https://kb.example/mcp" {
		t.Fatalf("resource %v", pr["resource"])
	}

	req = httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	req.Host = "kb.example"
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var as map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &as); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"authorization_endpoint", "token_endpoint", "registration_endpoint"} {
		if as[key] == nil || as[key] == "" {
			t.Fatalf("missing %s", key)
		}
	}
	methods, _ := as["code_challenge_methods_supported"].([]any)
	if len(methods) != 1 || methods[0] != "S256" {
		t.Fatalf("pkce methods %v", methods)
	}
}

func TestSharedMCPRejectsEndpointToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	oauth := handler.NewSharedMCPOAuthHandler(stubSharedMCPService{})
	srv := mcpserver.NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	RegisterSharedMCPRoutes(r, oauth, srv, stubSharedMCPService{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Host = "kb.example"
	req.Header.Set("Authorization", "Bearer "+types.MCPEndpointTokenPrefix+"deadbeef")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}
