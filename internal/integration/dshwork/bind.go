package dshwork

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	bindRedirectAllowlistEnv = "DSH_WORK_BIND_REDIRECT_ALLOWLIST"
	bindSessionCookieName    = "dsh_work_bind_session"
	bindTTL                  = 10 * time.Minute
)

// BindSessionCookieName is the HttpOnly cookie used during browser bind login.
const BindSessionCookieName = bindSessionCookieName

type pendingBind struct {
	redirectURI string
	state       string
	expiresAt   time.Time
}

type issuedBindCode struct {
	userID    string
	expiresAt time.Time
	used      bool
}

var (
	bindMu          sync.Mutex
	pendingByToken  = map[string]pendingBind{}
	codesByHash     = map[string]issuedBindCode{}
)

func resetBindStoreForTest() {
	bindMu.Lock()
	defer bindMu.Unlock()
	pendingByToken = map[string]pendingBind{}
	codesByHash = map[string]issuedBindCode{}
}

// ResetBindStoreForTest clears in-memory bind state (tests only).
func ResetBindStoreForTest() { resetBindStoreForTest() }

func redirectAllowlist() []string {
	raw := strings.TrimSpace(os.Getenv(bindRedirectAllowlistEnv))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func redirectAllowed(redirectURI string) bool {
	redirectURI = strings.TrimSpace(redirectURI)
	if redirectURI == "" {
		return false
	}
	for _, allowed := range redirectAllowlist() {
		if redirectURI == allowed {
			return true
		}
	}
	return false
}

func newOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashBindCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func purgeExpiredLocked(now time.Time) {
	for k, v := range pendingByToken {
		if now.After(v.expiresAt) {
			delete(pendingByToken, k)
		}
	}
	for k, v := range codesByHash {
		if now.After(v.expiresAt) {
			delete(codesByHash, k)
		}
	}
}

// StartBind validates redirect_uri and registers a pending browser bind session.
// enterPath should be the path-only enter URL (handler adds host).
func StartBind(redirectURI, state string) (enterPath string, sessionToken string, err error) {
	if !redirectAllowed(redirectURI) {
		return "", "", ErrRedirectNotAllowed
	}
	state = strings.TrimSpace(state)
	if state == "" {
		return "", "", ErrRedirectNotAllowed
	}
	token, err := newOpaqueToken()
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	bindMu.Lock()
	defer bindMu.Unlock()
	purgeExpiredLocked(now)
	pendingByToken[token] = pendingBind{
		redirectURI: strings.TrimSpace(redirectURI),
		state:       state,
		expiresAt:   now.Add(bindTTL),
	}
	return "/api/v1/integrations/dsh-work/bind/enter?token=" + token, token, nil
}

// LookupPendingBind returns redirect_uri and state for a session token.
func LookupPendingBind(sessionToken string) (redirectURI, state string, ok bool) {
	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return "", "", false
	}
	now := time.Now()
	bindMu.Lock()
	defer bindMu.Unlock()
	purgeExpiredLocked(now)
	pending, exists := pendingByToken[sessionToken]
	if !exists || now.After(pending.expiresAt) {
		return "", "", false
	}
	return pending.redirectURI, pending.state, true
}

// FailPendingBind redirects to bind_error when login was denied or failed.
func FailPendingBind(sessionToken string) (redirectURL string, ok bool) {
	redirectURI, state, found := LookupPendingBind(sessionToken)
	if !found {
		return "", false
	}
	deletePending(sessionToken)
	return redirectURI + "?state=" + urlQueryEscape(state) + "&bind_error=access_denied", true
}

func deletePending(sessionToken string) {
	bindMu.Lock()
	defer bindMu.Unlock()
	delete(pendingByToken, sessionToken)
}

// CompletePendingBind issues a one-time bind_code and returns the callback URL.
func CompletePendingBind(sessionToken, userID string) (redirectURL string, ok bool) {
	sessionToken = strings.TrimSpace(sessionToken)
	userID = strings.TrimSpace(userID)
	if sessionToken == "" || userID == "" {
		return "", false
	}
	redirectURI, state, found := LookupPendingBind(sessionToken)
	if !found {
		return "", false
	}
	code, err := newOpaqueToken()
	if err != nil {
		return "", false
	}
	now := time.Now()
	bindMu.Lock()
	delete(pendingByToken, sessionToken)
	codesByHash[hashBindCode(code)] = issuedBindCode{
		userID:    userID,
		expiresAt: now.Add(bindTTL),
		used:      false,
	}
	bindMu.Unlock()
	return redirectURI + "?state=" + urlQueryEscape(state) + "&bind_code=" + urlQueryEscape(code), true
}

// ExchangeBindCode validates a bind code and returns the bound user id once.
func ExchangeBindCode(code string) (userID string, err error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", ErrBindCodeInvalid
	}
	h := hashBindCode(code)
	now := time.Now()
	bindMu.Lock()
	defer bindMu.Unlock()
	purgeExpiredLocked(now)
	row, ok := codesByHash[h]
	if !ok || now.After(row.expiresAt) || row.used {
		return "", ErrBindCodeInvalid
	}
	row.used = true
	codesByHash[h] = row
	return row.userID, nil
}

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		" ", "%20",
		"#", "%23",
		"&", "%26",
		"+", "%2B",
		"=", "%3D",
		"?", "%3F",
	)
	return replacer.Replace(value)
}
