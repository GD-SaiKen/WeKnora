package dshwork

import (
	"crypto/subtle"
	"os"
	"strings"
)

const integrationSecretEnv = "DSH_WORK_INTEGRATION_SECRET"

// Configured reports whether the integration secret is set.
func Configured() bool {
	return strings.TrimSpace(os.Getenv(integrationSecretEnv)) != ""
}

// configuredSecret returns the trimmed secret or empty when unset.
func configuredSecret() string {
	return strings.TrimSpace(os.Getenv(integrationSecretEnv))
}

// ValidateBearer compares the Authorization bearer value to the configured secret.
func ValidateBearer(bearer string) error {
	if !Configured() {
		return ErrNotConfigured
	}
	expected := configuredSecret()
	token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(bearer), "Bearer "))
	if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		return ErrUnauthorizedSecret
	}
	return nil
}
