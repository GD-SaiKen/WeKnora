package dshwork

import (
	"os"
	"testing"
)

func TestSecretConfigured(t *testing.T) {
	t.Setenv(integrationSecretEnv, "")
	if Configured() {
		t.Fatal("expected not configured")
	}
	t.Setenv(integrationSecretEnv, "  sekret  ")
	if !Configured() {
		t.Fatal("expected configured")
	}
}

func TestSecretValidateBearer(t *testing.T) {
	t.Setenv(integrationSecretEnv, "")
	if err := ValidateBearer("Bearer x"); err != ErrNotConfigured {
		t.Fatalf("got %v want ErrNotConfigured", err)
	}

	t.Setenv(integrationSecretEnv, "expected")
	if err := ValidateBearer("Bearer wrong"); err != ErrUnauthorizedSecret {
		t.Fatalf("got %v want ErrUnauthorizedSecret", err)
	}
	if err := ValidateBearer("expected"); err != ErrUnauthorizedSecret {
		t.Fatalf("bare token should fail: %v", err)
	}
	if err := ValidateBearer("Bearer expected"); err != nil {
		t.Fatalf("valid secret: %v", err)
	}
}

func TestSecretUsesEnvName(t *testing.T) {
	if integrationSecretEnv != "DSH_WORK_INTEGRATION_SECRET" {
		t.Fatalf("unexpected env constant %q", integrationSecretEnv)
	}
	_ = os.Getenv(integrationSecretEnv)
}
