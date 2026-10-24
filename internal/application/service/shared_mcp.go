package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/sharedmcp"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

type sharedMCPService struct {
	repo          interfaces.SharedMCPRepository
	tenantMembers interfaces.TenantMemberService
	users         interfaces.UserService
	tenants       interfaces.TenantService
}

// NewSharedMCPService creates the shared /mcp OAuth service.
func NewSharedMCPService(
	repo interfaces.SharedMCPRepository,
	tenantMembers interfaces.TenantMemberService,
	users interfaces.UserService,
	tenants interfaces.TenantService,
) interfaces.SharedMCPService {
	return &sharedMCPService{
		repo:          repo,
		tenantMembers: tenantMembers,
		users:         users,
		tenants:       tenants,
	}
}

func (s *sharedMCPService) GetConfig(ctx context.Context, tenantID uint64) (*types.SharedMCPConfig, error) {
	return s.repo.GetConfig(ctx, tenantID)
}

func (s *sharedMCPService) UpdateConfig(ctx context.Context, tenantID uint64, cfg *types.SharedMCPConfig) (*types.SharedMCPConfig, error) {
	if cfg == nil {
		return nil, apperrors.NewBadRequestError("config is required")
	}
	tools := types.NormalizeMCPEndpointTools(cfg.Tools)
	if len(tools) == 0 {
		tools = types.DefaultSharedMCPTools()
	}
	for _, name := range tools {
		if _, ok := types.LookupMCPEndpointTool(name); !ok {
			return nil, apperrors.NewBadRequestError("unknown tool: " + name)
		}
	}
	rate := cfg.RateLimitPerMinute
	if rate <= 0 {
		rate = types.DefaultMCPEndpointRateLimitPerMinute
	}
	if rate > mcpEndpointMaxRatePerMin {
		return nil, apperrors.NewBadRequestError("rate limit is too high")
	}
	out := &types.SharedMCPConfig{
		TenantID:             tenantID,
		Enabled:              cfg.Enabled,
		Tools:                types.StringArray(tools),
		KnowledgeBaseIDs:     cfg.KnowledgeBaseIDs,
		DefaultAgentID:       strings.TrimSpace(cfg.DefaultAgentID),
		RateLimitPerMinute:   rate,
		RedirectURIAllowlist: cfg.RedirectURIAllowlist,
	}
	if err := s.repo.UpsertConfig(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *sharedMCPService) RegisterClient(ctx context.Context, name string, redirectURIs []string) (*types.SharedMCPClient, error) {
	if len(redirectURIs) == 0 {
		return nil, apperrors.NewBadRequestError("redirect_uris is required")
	}
	for _, uri := range redirectURIs {
		if !s.redirectURIGloballyAllowed(ctx, uri) {
			return nil, apperrors.NewBadRequestError("redirect uri is not allowed")
		}
	}
	client := &types.SharedMCPClient{
		Name:         strings.TrimSpace(name),
		RedirectURIs: types.StringArray(redirectURIs),
	}
	if err := s.repo.CreateClient(ctx, client); err != nil {
		return nil, err
	}
	return client, nil
}

func (s *sharedMCPService) AuthorizeTenants(ctx context.Context, userID string) ([]uint64, error) {
	members, err := s.tenantMembers.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	enabled, err := s.repo.ListEnabledTenantIDs(ctx)
	if err != nil {
		return nil, err
	}
	enabledSet := make(map[uint64]bool, len(enabled))
	for _, id := range enabled {
		enabledSet[id] = true
	}
	var out []uint64
	for _, m := range members {
		if m == nil || !enabledSet[m.TenantID] {
			continue
		}
		out = append(out, m.TenantID)
	}
	return out, nil
}

func (s *sharedMCPService) IssueAuthorizationCode(
	ctx context.Context,
	userID string,
	tenantID uint64,
	clientID, redirectURI, codeChallenge string,
	now time.Time,
) (string, error) {
	if err := s.ensureMember(ctx, userID, tenantID); err != nil {
		return "", err
	}
	cfg, err := s.requireEnabledConfig(ctx, tenantID)
	if err != nil {
		return "", err
	}
	client, err := s.repo.GetClient(ctx, clientID)
	if client == nil || err != nil {
		return "", apperrors.NewBadRequestError("unknown client")
	}
	if !clientRedirectAllowed(client, redirectURI) {
		return "", apperrors.NewBadRequestError("redirect uri mismatch")
	}
	if !s.tenantRedirectAllowed(cfg, redirectURI) && !isLoopbackRedirect(redirectURI) {
		return "", apperrors.NewBadRequestError("redirect uri is not allowed")
	}
	codeChallenge = strings.TrimSpace(codeChallenge)
	if codeChallenge == "" {
		return "", apperrors.NewBadRequestError("code_challenge is required")
	}
	plain, err := randomURLToken(32)
	if err != nil {
		return "", err
	}
	row := &types.SharedMCPAuthCode{
		CodeHash:      HashMCPEndpointToken(plain),
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		CodeChallenge: codeChallenge,
		UserID:        userID,
		TenantID:      tenantID,
		ExpiresAt:     now.Add(10 * time.Minute),
	}
	if err := s.repo.CreateAuthCode(ctx, row); err != nil {
		return "", err
	}
	return plain, nil
}

func (s *sharedMCPService) ExchangeAuthorizationCode(
	ctx context.Context,
	code, redirectURI, clientID, codeVerifier string,
	now time.Time,
) (string, string, int, error) {
	row, err := s.repo.ConsumeAuthCode(ctx, HashMCPEndpointToken(code), now)
	if row == nil || err != nil {
		return "", "", 0, apperrors.NewBadRequestError("invalid authorization code")
	}
	if row.ClientID != clientID || row.RedirectURI != redirectURI {
		return "", "", 0, apperrors.NewBadRequestError("invalid authorization code")
	}
	if err := sharedmcp.VerifyS256(row.CodeChallenge, codeVerifier); err != nil {
		return "", "", 0, apperrors.NewBadRequestError("invalid pkce verifier")
	}
	return s.issueForGrant(ctx, row.UserID, row.TenantID, row.ClientID, now)
}

func (s *sharedMCPService) Refresh(ctx context.Context, refreshToken string, now time.Time) (string, string, int, error) {
	claims, err := sharedmcp.ParseRefreshToken(refreshToken, getJwtSecret(), now)
	if err != nil {
		return "", "", 0, apperrors.NewUnauthorizedError("invalid refresh token")
	}
	grant, err := s.repo.GetGrant(ctx, claims.GrantID)
	if grant == nil || err != nil {
		return "", "", 0, apperrors.NewUnauthorizedError("invalid refresh token")
	}
	if grant.RevokedAt != nil {
		return "", "", 0, apperrors.NewUnauthorizedError("refresh token revoked")
	}
	if !grant.RefreshExpiresAt.After(now) || !grant.AbsoluteExpiresAt.After(now) {
		return "", "", 0, apperrors.NewUnauthorizedError("refresh token expired")
	}
	if HashMCPEndpointToken(refreshToken) != grant.RefreshTokenHash {
		return "", "", 0, apperrors.NewUnauthorizedError("invalid refresh token")
	}
	cfg, err := s.requireEnabledConfig(ctx, grant.TenantID)
	if err != nil {
		return "", "", 0, err
	}
	_ = cfg
	access, newRefresh, expiresIn, err := s.signPair(grant, now)
	if err != nil {
		return "", "", 0, err
	}
	newExp := now.Add(sharedmcp.RefreshTokenLifetime)
	if newExp.After(grant.AbsoluteExpiresAt) {
		newExp = grant.AbsoluteExpiresAt
	}
	if err := s.repo.UpdateGrantRefresh(ctx, grant.ID, HashMCPEndpointToken(newRefresh), newExp); err != nil {
		return "", "", 0, err
	}
	return access, newRefresh, expiresIn, nil
}

func (s *sharedMCPService) RevokeGrant(ctx context.Context, userID, grantID string, now time.Time) error {
	grant, err := s.repo.GetGrant(ctx, grantID)
	if grant == nil || err != nil {
		return apperrors.NewNotFoundError("grant not found")
	}
	if grant.UserID != userID {
		return apperrors.NewForbiddenError("grant not found")
	}
	return s.repo.RevokeGrant(ctx, grantID, now)
}

func (s *sharedMCPService) ListGrants(ctx context.Context, userID string) ([]*types.SharedMCPGrant, error) {
	return s.repo.ListGrantsByUser(ctx, userID)
}

func (s *sharedMCPService) ResolveAccessToken(ctx context.Context, accessToken string, now time.Time) (*types.SharedMCPCall, error) {
	claims, err := sharedmcp.ParseAccessToken(accessToken, getJwtSecret(), now)
	if err != nil {
		return nil, err
	}
	grant, err := s.repo.GetGrant(ctx, claims.GrantID)
	if grant == nil || err != nil {
		return nil, errors.New("grant not found")
	}
	if grant.RevokedAt != nil || grant.UserID != claims.UserID || grant.TenantID != claims.TenantID {
		return nil, errors.New("grant not valid")
	}
	cfg, err := s.requireEnabledConfig(ctx, claims.TenantID)
	if err != nil {
		return nil, err
	}
	user, err := s.users.GetUserByID(ctx, claims.UserID)
	if user == nil || err != nil || !user.IsActive {
		return nil, errors.New("user not active")
	}
	member, err := s.tenantMembers.GetMembership(ctx, claims.UserID, claims.TenantID)
	if member == nil || err != nil {
		return nil, errors.New("not a workspace member")
	}
	tenant, err := s.tenants.GetTenantByID(ctx, claims.TenantID)
	if tenant == nil || err != nil {
		return nil, errors.New("workspace unavailable")
	}
	if strings.EqualFold(tenant.Status, "inactive") || strings.EqualFold(tenant.Status, "disabled") {
		return nil, errors.New("workspace inactive")
	}
	tools := []string(cfg.Tools)
	if len(tools) == 0 {
		tools = types.DefaultSharedMCPTools()
	}
	return &types.SharedMCPCall{
		User:               user,
		TenantID:           claims.TenantID,
		Role:               member.Role,
		Tools:              tools,
		KnowledgeBaseIDs:   []string(cfg.KnowledgeBaseIDs),
		DefaultAgentID:     cfg.DefaultAgentID,
		RateLimitPerMinute: cfg.RateLimitPerMinute,
		ClientID:           claims.ClientID,
		GrantID:            claims.GrantID,
	}, nil
}

func (s *sharedMCPService) RedirectURIAllowed(ctx context.Context, tenantID uint64, redirectURI string) bool {
	if isLoopbackRedirect(redirectURI) {
		return true
	}
	cfg, err := s.repo.GetConfig(ctx, tenantID)
	if cfg == nil || err != nil {
		return false
	}
	return s.tenantRedirectAllowed(cfg, redirectURI)
}

func (s *sharedMCPService) issueForGrant(
	ctx context.Context,
	userID string,
	tenantID uint64,
	clientID string,
	now time.Time,
) (string, string, int, error) {
	if err := s.ensureMember(ctx, userID, tenantID); err != nil {
		return "", "", 0, err
	}
	if _, err := s.requireEnabledConfig(ctx, tenantID); err != nil {
		return "", "", 0, err
	}
	grant := &types.SharedMCPGrant{
		ID:                uuid.New().String(),
		ClientID:          clientID,
		UserID:            userID,
		TenantID:          tenantID,
		RefreshExpiresAt:  now.Add(sharedmcp.RefreshTokenLifetime),
		AbsoluteExpiresAt: now.Add(sharedmcp.GrantAbsoluteLifetime),
	}
	access, refresh, expiresIn, err := s.signPair(grant, now)
	if err != nil {
		return "", "", 0, err
	}
	grant.RefreshTokenHash = HashMCPEndpointToken(refresh)
	if err := s.repo.CreateGrant(ctx, grant); err != nil {
		return "", "", 0, err
	}
	return access, refresh, expiresIn, nil
}

func (s *sharedMCPService) signPair(grant *types.SharedMCPGrant, now time.Time) (string, string, int, error) {
	access, refresh, err := sharedmcp.IssueTokenPair(sharedmcp.TokenInput{
		UserID:   grant.UserID,
		TenantID: grant.TenantID,
		ClientID: grant.ClientID,
		GrantID:  grant.ID,
		Now:      now,
	}, getJwtSecret())
	if err != nil {
		return "", "", 0, err
	}
	return access, refresh, int(sharedmcp.AccessTokenLifetime.Seconds()), nil
}

func (s *sharedMCPService) ensureMember(ctx context.Context, userID string, tenantID uint64) error {
	member, err := s.tenantMembers.GetMembership(ctx, userID, tenantID)
	if member == nil || err != nil {
		return apperrors.NewForbiddenError("not a workspace member")
	}
	return nil
}

func (s *sharedMCPService) requireEnabledConfig(ctx context.Context, tenantID uint64) (*types.SharedMCPConfig, error) {
	cfg, err := s.repo.GetConfig(ctx, tenantID)
	if cfg == nil || err != nil || !cfg.Enabled {
		return nil, apperrors.NewForbiddenError("shared mcp is not enabled for this workspace")
	}
	return cfg, nil
}

func (s *sharedMCPService) redirectURIGloballyAllowed(ctx context.Context, redirectURI string) bool {
	if isLoopbackRedirect(redirectURI) {
		return true
	}
	ids, err := s.repo.ListEnabledTenantIDs(ctx)
	if err != nil {
		return false
	}
	for _, id := range ids {
		cfg, err := s.repo.GetConfig(ctx, id)
		if cfg != nil && err == nil && s.tenantRedirectAllowed(cfg, redirectURI) {
			return true
		}
	}
	return false
}

func (s *sharedMCPService) tenantRedirectAllowed(cfg *types.SharedMCPConfig, redirectURI string) bool {
	for _, allowed := range cfg.RedirectURIAllowlist {
		if strings.TrimSpace(allowed) == redirectURI {
			return true
		}
	}
	return false
}

func clientRedirectAllowed(client *types.SharedMCPClient, redirectURI string) bool {
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			return true
		}
	}
	return false
}

func isLoopbackRedirect(redirectURI string) bool {
	u, err := url.Parse(strings.TrimSpace(redirectURI))
	if err != nil || u.Scheme != "http" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "127.0.0.1" || host == "localhost" || host == "[::1]"
}

func randomURLToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
