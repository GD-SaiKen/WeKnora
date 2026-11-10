package dshwork

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

type fakeUserReader struct {
	users map[string]*types.User
}

func (f *fakeUserReader) GetUserByID(_ context.Context, id string) (*types.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

type fakeDirectory struct {
	byAIHub   map[string]string
	bySubject map[string]string
}

func (f *fakeDirectory) GetUserIDByAIHubUserID(_ context.Context, aiHubUserID string) (string, error) {
	return f.byAIHub[aiHubUserID], nil
}

func (f *fakeDirectory) GetUserIDBySubject(_ context.Context, subject string) (string, error) {
	return f.bySubject[subject], nil
}

func activeUser(id string) *types.User {
	return &types.User{ID: id, TenantID: 42, IsActive: true}
}

func TestSessionResolveByAIHub(t *testing.T) {
	ctx := context.Background()
	resolver := &SessionResolver{
		Users:     &fakeUserReader{users: map[string]*types.User{"u1": activeUser("u1")}},
		Directory: &fakeDirectory{byAIHub: map[string]string{"hub-1": "u1"}},
	}
	u, err := resolver.ResolveUser(ctx, SessionExchangeRequest{AIHubUserID: "hub-1"})
	if err != nil || u.ID != "u1" {
		t.Fatalf("resolve: user=%v err=%v", u, err)
	}
}

func TestSessionResolveBySubject(t *testing.T) {
	ctx := context.Background()
	resolver := &SessionResolver{
		Users:     &fakeUserReader{users: map[string]*types.User{"u2": activeUser("u2")}},
		Directory: &fakeDirectory{bySubject: map[string]string{"sub-1": "u2"}},
	}
	u, err := resolver.ResolveUser(ctx, SessionExchangeRequest{Subject: "sub-1"})
	if err != nil || u.ID != "u2" {
		t.Fatalf("resolve: user=%v err=%v", u, err)
	}
}

func TestSessionAIHubWinsOverSubject(t *testing.T) {
	ctx := context.Background()
	resolver := &SessionResolver{
		Users: &fakeUserReader{
			users: map[string]*types.User{
				"u-ai":  activeUser("u-ai"),
				"u-sub": activeUser("u-sub"),
			},
		},
		Directory: &fakeDirectory{
			byAIHub:   map[string]string{"hub": "u-ai"},
			bySubject: map[string]string{"sub": "u-sub"},
		},
	}
	u, err := resolver.ResolveUser(ctx, SessionExchangeRequest{AIHubUserID: "hub", Subject: "sub"})
	if err != nil || u.ID != "u-ai" {
		t.Fatalf("expected ai hub user, got %v err=%v", u, err)
	}
}

func TestSessionNotProvisioned(t *testing.T) {
	ctx := context.Background()
	resolver := &SessionResolver{
		Users:     &fakeUserReader{users: map[string]*types.User{}},
		Directory: &fakeDirectory{},
	}
	_, err := resolver.ResolveUser(ctx, SessionExchangeRequest{Subject: "missing"})
	if err != ErrAccountNotProvisioned {
		t.Fatalf("got %v", err)
	}
}

func TestSessionDisabled(t *testing.T) {
	ctx := context.Background()
	disabled := activeUser("u3")
	disabled.IsActive = false
	resolver := &SessionResolver{
		Users:     &fakeUserReader{users: map[string]*types.User{"u3": disabled}},
		Directory: &fakeDirectory{bySubject: map[string]string{"s": "u3"}},
	}
	_, err := resolver.ResolveUser(ctx, SessionExchangeRequest{Subject: "s"})
	if err != ErrAccountDisabled {
		t.Fatalf("got %v", err)
	}
}

func TestSessionByUserID(t *testing.T) {
	ctx := context.Background()
	resolver := &SessionResolver{
		Users: &fakeUserReader{users: map[string]*types.User{"bound": activeUser("bound")}},
	}
	u, err := resolver.ResolveUser(ctx, SessionExchangeRequest{UserID: "bound"})
	if err != nil || u.ID != "bound" {
		t.Fatalf("resolve: user=%v err=%v", u, err)
	}
	_, err = resolver.ResolveUser(ctx, SessionExchangeRequest{UserID: "ghost"})
	if err != ErrAccountNotProvisioned {
		t.Fatalf("got %v", err)
	}
}

func TestSessionEmptyRequest(t *testing.T) {
	ctx := context.Background()
	resolver := &SessionResolver{Users: &fakeUserReader{users: map[string]*types.User{}}}
	_, err := resolver.ResolveUser(ctx, SessionExchangeRequest{})
	if err != ErrInvalidSessionRequest {
		t.Fatalf("got %v", err)
	}
}
