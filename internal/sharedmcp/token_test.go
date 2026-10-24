package sharedmcp

import (
	"testing"
	"time"
)

func TestMCPAccessTokenIsNotAWebToken(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	access, refresh, err := IssueTokenPair(TokenInput{
		UserID: "user-1", TenantID: 7, ClientID: "client-1", GrantID: "grant-1", Now: now,
	}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseAccessToken(access, "test-secret", now)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Audience != AccessAudience || claims.UserID != "user-1" || claims.TenantID != 7 {
		t.Fatalf("claims %+v", claims)
	}
	if claims.ExpiresAt.Sub(claims.IssuedAt) != AccessTokenLifetime {
		t.Fatalf("access ttl %s", claims.ExpiresAt.Sub(claims.IssuedAt))
	}
	rc, err := ParseRefreshToken(refresh, "test-secret", now)
	if err != nil || rc.GrantID != "grant-1" {
		t.Fatalf("refresh %+v %v", rc, err)
	}
}
