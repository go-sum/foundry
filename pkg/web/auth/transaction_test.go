package auth

import (
	"testing"
	"time"
)

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

func TestNewTransaction_SetsCreatedAt(t *testing.T) {
	before := time.Now().UTC()
	tx, err := NewTransaction("/")
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("NewTransaction error: %v", err)
	}
	if tx.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero, want non-zero")
	}
	if tx.CreatedAt.Before(before) || tx.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, want between %v and %v", tx.CreatedAt, before, after)
	}
}
