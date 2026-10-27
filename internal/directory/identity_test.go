package directory

import "testing"

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Zhou@Company.com "); got != "zhou@company.com" {
		t.Fatalf("got %q", got)
	}
}
