package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

// SharedMCPRepository persists shared /mcp configuration and OAuth state.
type SharedMCPRepository interface {
	GetConfig(ctx context.Context, tenantID uint64) (*types.SharedMCPConfig, error)
	UpsertConfig(ctx context.Context, cfg *types.SharedMCPConfig) error
	CreateClient(ctx context.Context, client *types.SharedMCPClient) error
	GetClient(ctx context.Context, id string) (*types.SharedMCPClient, error)
	CreateAuthCode(ctx context.Context, row *types.SharedMCPAuthCode) error
	ConsumeAuthCode(ctx context.Context, codeHash string, now time.Time) (*types.SharedMCPAuthCode, error)
	CreateGrant(ctx context.Context, grant *types.SharedMCPGrant) error
	GetGrant(ctx context.Context, id string) (*types.SharedMCPGrant, error)
	UpdateGrantRefresh(ctx context.Context, grantID, refreshHash string, refreshExpiresAt time.Time) error
	RevokeGrant(ctx context.Context, grantID string, now time.Time) error
	ListGrantsByUser(ctx context.Context, userID string) ([]*types.SharedMCPGrant, error)
	ListEnabledTenantIDs(ctx context.Context) ([]uint64, error)
}

// SharedMCPService implements OAuth and configuration for the fixed /mcp entry.
type SharedMCPService interface {
	GetConfig(ctx context.Context, tenantID uint64) (*types.SharedMCPConfig, error)
	UpdateConfig(ctx context.Context, tenantID uint64, cfg *types.SharedMCPConfig) (*types.SharedMCPConfig, error)
	RegisterClient(ctx context.Context, name string, redirectURIs []string) (*types.SharedMCPClient, error)
	AuthorizeTenants(ctx context.Context, userID string) ([]uint64, error)
	IssueAuthorizationCode(
		ctx context.Context,
		userID string,
		tenantID uint64,
		clientID, redirectURI, codeChallenge string,
		now time.Time,
	) (string, error)
	ExchangeAuthorizationCode(
		ctx context.Context,
		code, redirectURI, clientID, codeVerifier string,
		now time.Time,
	) (accessToken, refreshToken string, expiresIn int, err error)
	Refresh(
		ctx context.Context,
		refreshToken string,
		now time.Time,
	) (accessToken, newRefresh string, expiresIn int, err error)
	RevokeGrant(ctx context.Context, userID, grantID string, now time.Time) error
	ListGrants(ctx context.Context, userID string) ([]*types.SharedMCPGrant, error)
	ResolveAccessToken(ctx context.Context, accessToken string, now time.Time) (*types.SharedMCPCall, error)
	RedirectURIAllowed(ctx context.Context, tenantID uint64, redirectURI string) bool
}
