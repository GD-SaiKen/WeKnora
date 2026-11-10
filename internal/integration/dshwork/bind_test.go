package dshwork

import (
	"strings"
	"testing"
	"time"
)

func TestBindRedirectNotAllowed(t *testing.T) {
	resetBindStoreForTest()
	t.Setenv(bindRedirectAllowlistEnv, "https://dsh.example/callback")
	_, _, err := StartBind("https://evil.example/callback", "state-1")
	if err != ErrRedirectNotAllowed {
		t.Fatalf("got %v", err)
	}
}

func TestBindStartAndExchange(t *testing.T) {
	resetBindStoreForTest()
	t.Setenv(bindRedirectAllowlistEnv, "https://dsh.example/callback")
	path, token, err := StartBind("https://dsh.example/callback", "opaque-state")
	if err != nil || token == "" || !strings.Contains(path, token) {
		t.Fatalf("start bind: path=%q token=%q err=%v", path, token, err)
	}

	redirect, ok := CompletePendingBind(token, "user-99")
	if !ok || !strings.Contains(redirect, "bind_code=") || !strings.Contains(redirect, "opaque-state") {
		t.Fatalf("complete bind redirect: %q ok=%v", redirect, ok)
	}
	idx := strings.Index(redirect, "bind_code=")
	codePart := redirect[idx+len("bind_code="):]
	code := strings.Split(codePart, "&")[0]
	userID, err := ExchangeBindCode(code)
	if err != nil || userID != "user-99" {
		t.Fatalf("exchange: user=%q err=%v", userID, err)
	}
	_, err = ExchangeBindCode(code)
	if err != ErrBindCodeInvalid {
		t.Fatalf("reuse should fail: %v", err)
	}
}

func TestBindExchangeWithoutLogin(t *testing.T) {
	resetBindStoreForTest()
	_, err := ExchangeBindCode("never-issued")
	if err != ErrBindCodeInvalid {
		t.Fatalf("got %v", err)
	}
}

func TestBindExpires(t *testing.T) {
	resetBindStoreForTest()
	t.Setenv(bindRedirectAllowlistEnv, "https://dsh.example/callback")
	_, token, err := StartBind("https://dsh.example/callback", "st")
	if err != nil {
		t.Fatal(err)
	}
	bindMu.Lock()
	pendingByToken[token] = pendingBind{
		redirectURI: "https://dsh.example/callback",
		state:       "st",
		expiresAt:   time.Now().Add(-time.Minute),
	}
	bindMu.Unlock()
	_, ok := CompletePendingBind(token, "u")
	if ok {
		t.Fatal("expected expired pending bind to fail")
	}
}
