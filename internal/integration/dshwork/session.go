package dshwork

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

// SessionExchangeRequest is the JSON body for integration session exchange.
type SessionExchangeRequest struct {
	Subject      string `json:"subject"`
	AIHubUserID  string `json:"ai_hub_user_id"`
	UserID       string `json:"user_id"`
}

// UserReader loads WeKnora users by primary key.
type UserReader interface {
	GetUserByID(ctx context.Context, id string) (*types.User, error)
}

// DirectoryReader resolves directory identities to user ids.
type DirectoryReader interface {
	GetUserIDByAIHubUserID(ctx context.Context, aiHubUserID string) (string, error)
	GetUserIDBySubject(ctx context.Context, subject string) (string, error)
}

// SessionResolver resolves integration identities to users without provisioning.
type SessionResolver struct {
	Users     UserReader
	Directory DirectoryReader
}

// ResolveUser finds an existing user or returns a domain error.
func (r *SessionResolver) ResolveUser(ctx context.Context, req SessionExchangeRequest) (*types.User, error) {
	if r == nil || r.Users == nil {
		return nil, ErrAccountNotProvisioned
	}

	subject := strings.TrimSpace(req.Subject)
	aiHub := strings.TrimSpace(req.AIHubUserID)
	userID := strings.TrimSpace(req.UserID)

	if subject == "" && aiHub == "" && userID == "" {
		return nil, ErrInvalidSessionRequest
	}

	var resolvedID string
	if aiHub != "" && r.Directory != nil {
		id, err := r.Directory.GetUserIDByAIHubUserID(ctx, aiHub)
		if err != nil {
			return nil, err
		}
		if id != "" {
			resolvedID = id
		}
	}
	if resolvedID == "" && subject != "" && r.Directory != nil {
		id, err := r.Directory.GetUserIDBySubject(ctx, subject)
		if err != nil {
			return nil, err
		}
		if id != "" {
			resolvedID = id
		}
	}
	if resolvedID == "" && userID != "" {
		resolvedID = userID
	}
	if resolvedID == "" {
		return nil, ErrAccountNotProvisioned
	}

	user, err := r.Users.GetUserByID(ctx, resolvedID)
	if err != nil || user == nil {
		return nil, ErrAccountNotProvisioned
	}
	if !user.IsActive {
		return nil, ErrAccountDisabled
	}
	return user, nil
}
