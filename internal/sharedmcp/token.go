package sharedmcp

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessAudience      = "weknora-mcp"
	AccessTokenType     = "mcp_access"
	RefreshTokenType    = "mcp_refresh"
	AccessTokenLifetime = time.Hour
	RefreshTokenLifetime = 30 * 24 * time.Hour
	GrantAbsoluteLifetime = 90 * 24 * time.Hour
)

// TokenInput carries the claims stored in a new token pair.
type TokenInput struct {
	UserID   string
	TenantID uint64
	ClientID string
	GrantID  string
	Now      time.Time
}

// AccessClaims are the validated MCP access token claims.
type AccessClaims struct {
	UserID    string
	TenantID  uint64
	ClientID  string
	GrantID   string
	Audience  string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// RefreshClaims are the validated MCP refresh token claims.
type RefreshClaims struct {
	GrantID   string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// IssueTokenPair signs a short-lived access token and a refresh token.
func IssueTokenPair(in TokenInput, secret string) (access string, refresh string, err error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", "", errors.New("missing signing secret")
	}
	if in.UserID == "" || in.ClientID == "" || in.GrantID == "" || in.TenantID == 0 {
		return "", "", errors.New("incomplete token input")
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	accessClaims := jwt.MapClaims{
		"aud":       AccessAudience,
		"type":      AccessTokenType,
		"user_id":   in.UserID,
		"tenant_id": in.TenantID,
		"client_id": in.ClientID,
		"grant_id":  in.GrantID,
		"iat":       now.Unix(),
		"exp":       now.Add(AccessTokenLifetime).Unix(),
	}
	accessTok := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	access, err = accessTok.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}
	refreshClaims := jwt.MapClaims{
		"type":     RefreshTokenType,
		"grant_id": in.GrantID,
		"iat":      now.Unix(),
		"exp":      now.Add(RefreshTokenLifetime).Unix(),
	}
	refreshTok := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refresh, err = refreshTok.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// ParseAccessToken validates an MCP access token.
func ParseAccessToken(tokenString, secret string, now time.Time) (*AccessClaims, error) {
	claims, err := parseClaims(tokenString, secret, now)
	if err != nil {
		return nil, err
	}
	typ, _ := claims["type"].(string)
	if typ != AccessTokenType {
		return nil, errors.New("not an mcp access token")
	}
	aud, err := audienceContains(claims, AccessAudience)
	if err != nil {
		return nil, err
	}
	userID, _ := claims["user_id"].(string)
	clientID, _ := claims["client_id"].(string)
	grantID, _ := claims["grant_id"].(string)
	tenantID, err := uint64Claim(claims, "tenant_id")
	if err != nil {
		return nil, err
	}
	if userID == "" || clientID == "" || grantID == "" || tenantID == 0 {
		return nil, errors.New("incomplete access token claims")
	}
	return &AccessClaims{
		UserID:    userID,
		TenantID:  tenantID,
		ClientID:  clientID,
		GrantID:   grantID,
		Audience:  aud,
		IssuedAt:  time.Unix(int64(claims["iat"].(float64)), 0),
		ExpiresAt: time.Unix(int64(claims["exp"].(float64)), 0),
	}, nil
}

// ParseRefreshToken validates an MCP refresh token.
func ParseRefreshToken(tokenString, secret string, now time.Time) (*RefreshClaims, error) {
	claims, err := parseClaims(tokenString, secret, now)
	if err != nil {
		return nil, err
	}
	typ, _ := claims["type"].(string)
	if typ != RefreshTokenType {
		return nil, errors.New("not an mcp refresh token")
	}
	grantID, _ := claims["grant_id"].(string)
	if grantID == "" {
		return nil, errors.New("missing grant_id")
	}
	return &RefreshClaims{
		GrantID:   grantID,
		IssuedAt:  time.Unix(int64(claims["iat"].(float64)), 0),
		ExpiresAt: time.Unix(int64(claims["exp"].(float64)), 0),
	}, nil
}

func parseClaims(tokenString, secret string, now time.Time) (jwt.MapClaims, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errors.New("missing signing secret")
	}
	if now.IsZero() {
		now = time.Now()
	}
	claims := jwt.MapClaims{}
	parser := jwt.NewParser(
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	exp, ok := claims["exp"].(float64)
	if !ok || now.Unix() >= int64(exp) {
		return nil, errors.New("token expired")
	}
	return claims, nil
}

func audienceContains(claims jwt.MapClaims, want string) (string, error) {
	raw := claims["aud"]
	switch v := raw.(type) {
	case string:
		if v != want {
			return "", errors.New("unexpected audience")
		}
		return v, nil
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == want {
				return want, nil
			}
		}
		return "", errors.New("unexpected audience")
	default:
		return "", errors.New("missing audience")
	}
}

// LooksLikeMCPSessionToken reports whether the bearer string is an MCP OAuth JWT.
func LooksLikeMCPSessionToken(tokenString string) bool {
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, claims)
	if err != nil {
		return false
	}
	typ, _ := claims["type"].(string)
	return typ == AccessTokenType || typ == RefreshTokenType
}

func uint64Claim(claims jwt.MapClaims, key string) (uint64, error) {
	v, ok := claims[key]
	if !ok {
		return 0, errors.New("missing claim")
	}
	switch n := v.(type) {
	case float64:
		if n <= 0 {
			return 0, errors.New("invalid claim")
		}
		return uint64(n), nil
	default:
		return 0, errors.New("invalid claim type")
	}
}
