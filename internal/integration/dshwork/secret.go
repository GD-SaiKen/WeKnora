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
//
// 只接受 "Bearer <token>" 形式：缺少 scheme 的裸 token 一律拒绝。
// （RFC 7235 规定 auth-scheme 不区分大小写，所以用 EqualFold 比较前缀。）
// 注意不要用 strings.TrimPrefix —— 它找不到前缀时原样返回，会让裸 token 直接
// 通过比较，等于绕过了 scheme 校验。
func ValidateBearer(bearer string) error {
	if !Configured() {
		return ErrNotConfigured
	}
	expected := configuredSecret()

	raw := strings.TrimSpace(bearer)
	const scheme = "Bearer "
	if len(raw) <= len(scheme) || !strings.EqualFold(raw[:len(scheme)], scheme) {
		return ErrUnauthorizedSecret
	}
	token := strings.TrimSpace(raw[len(scheme):])
	if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		return ErrUnauthorizedSecret
	}
	return nil
}
