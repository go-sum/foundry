package auth

import (
	"reflect"
	"testing"
)

func TestEffectiveScopes_NilScopes(t *testing.T) {
	cfg := ProviderConfig{Scopes: nil}
	got := cfg.EffectiveScopes()
	want := []string{"openid", "email", "profile"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("EffectiveScopes nil = %v, want %v", got, want)
	}
}

func TestEffectiveScopes_EmptyScopes(t *testing.T) {
	cfg := ProviderConfig{Scopes: []string{}}
	got := cfg.EffectiveScopes()
	want := []string{"openid", "email", "profile"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("EffectiveScopes empty = %v, want %v", got, want)
	}
}

func TestEffectiveScopes_ConfiguredScopes(t *testing.T) {
	cfg := ProviderConfig{Scopes: []string{"openid", "email"}}
	got := cfg.EffectiveScopes()
	want := []string{"openid", "email"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("EffectiveScopes configured = %v, want %v", got, want)
	}
}
