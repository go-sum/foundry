package auth

import "testing"

func TestNewTransaction_Fields(t *testing.T) {
	tx, err := NewTransaction("/dashboard")
	if err != nil {
		t.Fatalf("NewTransaction error: %v", err)
	}
	if tx.State == "" {
		t.Error("State is empty, want non-empty")
	}
	if tx.Nonce == "" {
		t.Error("Nonce is empty, want non-empty")
	}
	if tx.Verifier == "" {
		t.Error("Verifier is empty, want non-empty")
	}
	if tx.ReturnTo != "/dashboard" {
		t.Errorf("ReturnTo = %q, want %q", tx.ReturnTo, "/dashboard")
	}
}

func TestNewTransaction_InvalidReturnTo(t *testing.T) {
	tx, err := NewTransaction("//evil.com")
	if err != nil {
		t.Fatalf("NewTransaction error: %v", err)
	}
	if tx.ReturnTo != "/" {
		t.Errorf("ReturnTo = %q, want %q", tx.ReturnTo, "/")
	}
}
