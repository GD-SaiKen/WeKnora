package mcpserver

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type captureSharedMCPAudit struct {
	mu      sync.Mutex
	entries []*types.AuditLog
}

func (c *captureSharedMCPAudit) Log(_ context.Context, entry *types.AuditLog) error {
	c.mu.Lock()
	c.entries = append(c.entries, entry)
	c.mu.Unlock()
	return nil
}

func (c *captureSharedMCPAudit) LogDenied(context.Context, *gin.Context, uint64, string, string, types.TenantRole) error {
	return nil
}

func (c *captureSharedMCPAudit) List(context.Context, uint64, *interfaces.AuditLogQuery) ([]*types.AuditLog, error) {
	return nil, nil
}

func (c *captureSharedMCPAudit) Purge(context.Context, int) (int64, error) { return 0, nil }

func (c *captureSharedMCPAudit) last() *types.AuditLog {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) == 0 {
		return nil
	}
	return c.entries[len(c.entries)-1]
}

func newTestSharedEngine(t *testing.T, call *types.SharedMCPCall, audit interfaces.AuditLogService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	srv := NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, audit)
	r := gin.New()
	inject := func(c *gin.Context) {
		if call != nil {
			ctx := context.WithValue(c.Request.Context(), types.SharedMCPContextKey, call)
			ctx = context.WithValue(ctx, types.TenantIDContextKey, call.TenantID)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
	r.POST("/mcp", inject, gin.WrapH(srv.Handler()))
	return r
}

func defaultSharedCall(role types.TenantRole) *types.SharedMCPCall {
	return &types.SharedMCPCall{
		User:               &types.User{ID: "user-a"},
		TenantID:           7,
		Role:               role,
		Tools:              types.DefaultSharedMCPTools(),
		RateLimitPerMinute: 100,
		ClientID:           "client-1",
		GrantID:            "grant-1",
	}
}

func TestSharedMCPToolsListRespectsCeiling(t *testing.T) {
	r := newTestSharedEngine(t, defaultSharedCall(types.TenantRoleViewer), nil)
	names := toolNames(t, rpc(t, r, "tools/list", map[string]any{}))
	sort.Strings(names)
	want := append([]string(nil), types.DefaultSharedMCPTools()...)
	sort.Strings(want)
	if len(names) != len(want) {
		t.Fatalf("tools = %v want %v", names, want)
	}
	for _, forbidden := range []string{types.MCPEndpointToolAsk, types.MCPEndpointToolAddDocument} {
		if contains(names, forbidden) {
			t.Fatalf("ceiling must not list %s", forbidden)
		}
	}
}

func TestSharedMCPCallOutsideCeilingIs32602(t *testing.T) {
	r := newTestSharedEngine(t, defaultSharedCall(types.TenantRoleAdmin), nil)
	resp := rpc(t, r, "tools/call", map[string]any{
		"name":      types.MCPEndpointToolAsk,
		"arguments": map[string]any{"question": "hi"},
	})
	err, _ := resp["error"].(map[string]any)
	if err["code"] != float64(-32602) || resp["result"] != nil {
		t.Fatalf("expected -32602, got %v", resp)
	}
}

func TestSharedMCPViewerDestructiveIsToolError(t *testing.T) {
	call := defaultSharedCall(types.TenantRoleViewer)
	call.Tools = append(call.Tools, types.MCPEndpointToolAddDocument)
	audit := &captureSharedMCPAudit{}
	r := newTestSharedEngine(t, call, audit)
	resp := rpc(t, r, "tools/call", map[string]any{
		"name":      types.MCPEndpointToolAddDocument,
		"arguments": map[string]any{"knowledge_base_id": "kb-1", "title": "t", "content": "c"},
	})
	result, _ := resp["result"].(map[string]any)
	if result == nil || result["isError"] != true {
		t.Fatalf("expected tool isError, got %v", resp)
	}
	time.Sleep(50 * time.Millisecond)
	entry := audit.last()
	if entry == nil {
		t.Fatal("expected audit row")
	}
	details := auditDetailsForTest(entry.Details)
	if details["allowed"] != false {
		t.Fatalf("allowed flag: %v", details)
	}
	raw, _ := json.Marshal(details)
	s := string(raw)
	for _, secret := range []string{"mcp_", "refresh", "code_challenge", "Bearer"} {
		if strings.Contains(s, secret) {
			t.Fatalf("audit leaked %q in %s", secret, s)
		}
	}
}

func TestSharedMCPRateLimitIsPerUser(t *testing.T) {
	callA := defaultSharedCall(types.TenantRoleViewer)
	callA.RateLimitPerMinute = 1
	callB := defaultSharedCall(types.TenantRoleViewer)
	callB.User = &types.User{ID: "user-b"}
	callB.RateLimitPerMinute = 1

	srv := NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	rA := gin.New()
	rA.POST("/mcp", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.SharedMCPContextKey, callA)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, gin.WrapH(srv.Handler()))
	rB := gin.New()
	rB.POST("/mcp", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.SharedMCPContextKey, callB)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, gin.WrapH(srv.Handler()))

	callRead := func(r *gin.Engine) string {
		resp := rpc(t, r, "tools/call", map[string]any{
			"name":      types.MCPEndpointToolReadDocument,
			"arguments": map[string]any{},
		})
		result, _ := resp["result"].(map[string]any)
		content, _ := result["content"].([]any)
		first, _ := content[0].(map[string]any)
		return first["text"].(string)
	}
	if text := callRead(rA); !strings.Contains(text, "knowledge_id is required") {
		t.Fatalf("user A first: %q", text)
	}
	if text := callRead(rA); !strings.Contains(text, "rate limit") {
		t.Fatalf("user A second should rate limit: %q", text)
	}
	if text := callRead(rB); !strings.Contains(text, "knowledge_id is required") {
		t.Fatalf("user B should not inherit A's limit: %q", text)
	}
}

func contains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}
