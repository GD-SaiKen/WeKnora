package types

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SharedMCPConfig is the per-workspace ceiling for the fixed /mcp entry.
// When enabled is false or no row exists, the workspace cannot be chosen at
// authorization time and calls with a token for that tenant are rejected.
type SharedMCPConfig struct {
	TenantID           uint64      `json:"tenant_id"            gorm:"primaryKey"`
	Enabled            bool        `json:"enabled"              gorm:"not null;default:false"`
	Tools              StringArray `json:"tools"                gorm:"type:jsonb;not null;default:'[]'"`
	KnowledgeBaseIDs   StringArray `json:"knowledge_base_ids"   gorm:"type:jsonb;not null;default:'[]'"`
	DefaultAgentID     string      `json:"default_agent_id"     gorm:"type:varchar(36);not null;default:''"`
	RateLimitPerMinute int         `json:"rate_limit_per_minute" gorm:"not null;default:60"`
	// RedirectURIAllowlist holds extra redirect URIs allowed at dynamic client
	// registration in addition to loopback hosts.
	RedirectURIAllowlist StringArray `json:"redirect_uri_allowlist" gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

// TableName returns the storage table.
func (SharedMCPConfig) TableName() string { return "shared_mcp_configs" }

// DefaultSharedMCPTools is the read-only ceiling new workspaces start with.
func DefaultSharedMCPTools() []string {
	return []string{
		MCPEndpointToolListKnowledgeBases,
		MCPEndpointToolSearchKnowledge,
		MCPEndpointToolReadDocument,
	}
}

// SharedMCPClient is a dynamically registered OAuth client for /mcp.
type SharedMCPClient struct {
	ID           string      `json:"id"            gorm:"type:varchar(36);primaryKey"`
	Name         string      `json:"name"          gorm:"type:varchar(255);not null;default:''"`
	RedirectURIs StringArray `json:"redirect_uris" gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// TableName returns the storage table.
func (SharedMCPClient) TableName() string { return "shared_mcp_clients" }

// BeforeCreate assigns the id before insert.
func (c *SharedMCPClient) BeforeCreate(_ *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.RedirectURIs == nil {
		c.RedirectURIs = StringArray{}
	}
	return nil
}

// SharedMCPAuthCode stores a one-time authorization code (hashed).
type SharedMCPAuthCode struct {
	CodeHash      string     `json:"-"                 gorm:"type:varchar(64);primaryKey"`
	ClientID      string     `json:"client_id"         gorm:"type:varchar(36);not null;index"`
	RedirectURI   string     `json:"redirect_uri"      gorm:"type:text;not null;default:''"`
	CodeChallenge string     `json:"code_challenge"    gorm:"type:varchar(128);not null;default:''"`
	UserID        string     `json:"user_id"           gorm:"type:varchar(36);not null;index"`
	TenantID      uint64     `json:"tenant_id"         gorm:"not null;index"`
	ExpiresAt     time.Time  `json:"expires_at"        gorm:"not null"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TableName returns the storage table.
func (SharedMCPAuthCode) TableName() string { return "shared_mcp_auth_codes" }

// SharedMCPGrant stores refresh-token state for a user/client/tenant triple.
type SharedMCPGrant struct {
	ID                 string     `json:"id"                   gorm:"type:varchar(36);primaryKey"`
	ClientID           string     `json:"client_id"            gorm:"type:varchar(36);not null;index"`
	UserID             string     `json:"user_id"                gorm:"type:varchar(36);not null;index"`
	TenantID           uint64     `json:"tenant_id"              gorm:"not null;index"`
	RefreshTokenHash   string     `json:"-"                      gorm:"type:varchar(64);not null;default:'';index"`
	RefreshExpiresAt   time.Time  `json:"refresh_expires_at"     gorm:"not null"`
	AbsoluteExpiresAt  time.Time  `json:"absolute_expires_at"    gorm:"not null"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// TableName returns the storage table.
func (SharedMCPGrant) TableName() string { return "shared_mcp_grants" }

// BeforeCreate assigns the id before insert.
func (g *SharedMCPGrant) BeforeCreate(_ *gorm.DB) error {
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	return nil
}

// SharedMCPContextKey carries SharedMCPCall on the request context for /mcp.
var SharedMCPContextKey ContextKey = "shared_mcp_call"

// SharedMCPCallFromContext returns the /mcp call scope authenticated for this request.
func SharedMCPCallFromContext(ctx context.Context) (*SharedMCPCall, bool) {
	call, ok := ctx.Value(SharedMCPContextKey).(*SharedMCPCall)
	return call, ok && call != nil
}

// SharedMCPCall is the authenticated principal for the fixed /mcp surface.
type SharedMCPCall struct {
	User               *User
	TenantID           uint64
	Role               TenantRole
	Tools              []string
	KnowledgeBaseIDs   []string
	DefaultAgentID     string
	RateLimitPerMinute int
	ClientID           string
	GrantID            string
}
