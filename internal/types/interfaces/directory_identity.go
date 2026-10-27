package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/directory"
)

// DirectoryIdentityRepository persists ai-hub directory bindings.
type DirectoryIdentityRepository interface {
	GetUserIDBySubject(ctx context.Context, subject string) (string, error)
	GetUserIDByEmailAlias(ctx context.Context, normalizedEmail string) (string, error)
	GetUserIDByAIHubUserID(ctx context.Context, aiHubUserID string) (string, error)
}

// DirectorySyncService runs employee directory reconciliation.
type DirectorySyncService interface {
	State(ctx context.Context) (*directory.SyncState, error)
	Sync(ctx context.Context, actorID string, full bool) (*directory.SyncState, error)
}
