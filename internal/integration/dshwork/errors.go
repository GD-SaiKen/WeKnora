package dshwork

import "errors"

var (
	// ErrNotConfigured means DSH_WORK_INTEGRATION_SECRET is unset.
	ErrNotConfigured = errors.New("dsh-work integration not configured")
	// ErrUnauthorizedSecret means the bearer secret did not match.
	ErrUnauthorizedSecret = errors.New("dsh-work integration secret mismatch")
	// ErrAccountNotProvisioned means no matching WeKnora user exists.
	ErrAccountNotProvisioned = errors.New("account_not_provisioned")
	// ErrAccountDisabled means the matched user is inactive.
	ErrAccountDisabled = errors.New("account_disabled")
	// ErrInvalidSessionRequest means no usable identity was provided.
	ErrInvalidSessionRequest = errors.New("invalid session request")
	// ErrRedirectNotAllowed means redirect_uri is outside the allowlist.
	ErrRedirectNotAllowed = errors.New("redirect_not_allowed")
	// ErrBindCodeInvalid means the bind code is unknown, expired, or reused.
	ErrBindCodeInvalid = errors.New("bind_code_invalid")
)
