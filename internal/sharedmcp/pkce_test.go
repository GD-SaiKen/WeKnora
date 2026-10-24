package sharedmcp

import "testing"

func TestPKCERejectsPlainAndWrongVerifier(t *testing.T) {
	if err := VerifyS256("not-s256", "abc"); err == nil {
		t.Fatal("plain method must fail")
	}
	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	challenge := S256Challenge(verifier)
	if err := VerifyS256(challenge, verifier); err != nil {
		t.Fatal(err)
	}
	if err := VerifyS256(challenge, verifier+"x"); err == nil {
		t.Fatal("wrong verifier must fail")
	}
}
