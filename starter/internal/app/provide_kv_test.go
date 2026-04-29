package app

import (
	"testing"

	configpkg "github.com/go-sum/foundry/config"
)

func TestNeedsKV_CookieProduction_True(t *testing.T) {
	cfg := &configpkg.Config{
		Env:          configpkg.Production,
		SessionStore: "cookie",
	}
	if !needsKV(cfg) {
		t.Fatal("needsKV(cookie, production) = false, want true")
	}
}

func TestNeedsKV_KVProduction_True(t *testing.T) {
	cfg := &configpkg.Config{
		Env:          configpkg.Production,
		SessionStore: "kv",
	}
	if !needsKV(cfg) {
		t.Fatal("needsKV(kv, production) = false, want true")
	}
}

func TestNeedsKV_MemoryTesting_False(t *testing.T) {
	cfg := &configpkg.Config{
		Env:          configpkg.Testing,
		SessionStore: "memory",
	}
	if needsKV(cfg) {
		t.Fatal("needsKV(memory, testing) = true, want false")
	}
}

func TestNeedsKV_EmptySessionStore_False(t *testing.T) {
	cfg := &configpkg.Config{
		Env:          configpkg.Testing,
		SessionStore: "",
	}
	if needsKV(cfg) {
		t.Fatal("needsKV(empty, testing) = true, want false")
	}
}

func TestNeedsKV_EmptySessionStore_Production_True(t *testing.T) {
	cfg := &configpkg.Config{
		Env:          configpkg.Production,
		SessionStore: "",
	}
	if !needsKV(cfg) {
		t.Fatal("needsKV(empty, production) = false, want true")
	}
}
