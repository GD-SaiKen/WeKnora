package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type dshWorkIntegrationUserService struct {
	interfaces.UserService
	users map[string]*types.User
}

func (s *dshWorkIntegrationUserService) GetUserByID(_ context.Context, id string) (*types.User, error) {
	u, ok := s.users[id]
	if !ok {
		return nil, context.Canceled
	}
	return u, nil
}

func (s *dshWorkIntegrationUserService) IssueDshWorkIntegrationAccessToken(
	_ context.Context,
	user *types.User,
) (string, uint64, int, error) {
	return "access-token", user.TenantID, 3600, nil
}

type dshWorkIntegrationDirectory struct {
	interfaces.DirectoryIdentityRepository
}

func newDshWorkIntegrationTestRouter(t *testing.T, h *DshWorkIntegrationHandler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/integrations/dsh-work/sessions", h.CreateSession)
	r.POST("/api/v1/integrations/dsh-work/bind-codes", h.StartBindCode)
	r.POST("/api/v1/integrations/dsh-work/bind-codes/exchange", h.ExchangeBindCode)
	return r
}

func TestDshWorkIntegrationNotConfigured(t *testing.T) {
	t.Setenv("DSH_WORK_INTEGRATION_SECRET", "")
	h := NewDshWorkIntegrationHandler(&dshWorkIntegrationUserService{}, &dshWorkIntegrationDirectory{})
	r := newDshWorkIntegrationTestRouter(t, h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/dsh-work/sessions", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("sessions status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDshWorkIntegrationSessionSuccess(t *testing.T) {
	t.Setenv("DSH_WORK_INTEGRATION_SECRET", "test-secret")
	h := NewDshWorkIntegrationHandler(
		&dshWorkIntegrationUserService{users: map[string]*types.User{
			"u1": {ID: "u1", TenantID: 7, IsActive: true},
		}},
		&dshWorkIntegrationDirectory{},
	)
	r := newDshWorkIntegrationTestRouter(t, h)

	body := []byte(`{"user_id":"u1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/dsh-work/sessions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-secret")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["refresh_token"] != nil {
		t.Fatalf("unexpected refresh_token in %v", payload)
	}
	if payload["access_token"] != "access-token" || payload["user_id"] != "u1" {
		t.Fatalf("payload=%v", payload)
	}
}

func TestDshWorkIntegrationBindExchangeInvalid(t *testing.T) {
	t.Setenv("DSH_WORK_INTEGRATION_SECRET", "test-secret")
	h := NewDshWorkIntegrationHandler(&dshWorkIntegrationUserService{}, &dshWorkIntegrationDirectory{})
	r := newDshWorkIntegrationTestRouter(t, h)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/integrations/dsh-work/bind-codes/exchange",
		bytes.NewReader([]byte(`{"bind_code":"nope"}`)),
	)
	req.Header.Set("Authorization", "Bearer test-secret")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
	var payload map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &payload)
	if payload["error"] != "bind_code_invalid" {
		t.Fatalf("payload=%v", payload)
	}
}
