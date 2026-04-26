package provider

import (
	"reflect"
	"testing"
)

func TestIsAllowedRedirectURI_ExactMatch(t *testing.T) {
	registered := []string{
		"https://app.example.com/callback",
		"https://app.example.com/other",
	}
	if !isAllowedRedirectURI(registered, "https://app.example.com/callback") {
		t.Error("isAllowedRedirectURI: exact match returned false, want true")
	}
}

func TestIsAllowedRedirectURI_NoMatch(t *testing.T) {
	registered := []string{
		"https://app.example.com/callback",
	}
	cases := []string{
		"https://evil.example.com/callback",
		"https://app.example.com/callback/extra",
		"https://app.example.com/callbackx",
		"",
		"https://app.example.com/callback?foo=bar",
	}
	for _, requested := range cases {
		if isAllowedRedirectURI(registered, requested) {
			t.Errorf("isAllowedRedirectURI: %q should not match registered URIs", requested)
		}
	}
}

func TestIsAllowedRedirectURI_EmptyRegistered(t *testing.T) {
	if isAllowedRedirectURI(nil, "https://app.example.com/callback") {
		t.Error("isAllowedRedirectURI: nil registered list returned true, want false")
	}
	if isAllowedRedirectURI([]string{}, "https://app.example.com/callback") {
		t.Error("isAllowedRedirectURI: empty registered list returned true, want false")
	}
}

func TestIsAllowedRedirectURI_MultipleRegistered(t *testing.T) {
	registered := []string{
		"https://app.example.com/cb1",
		"https://app.example.com/cb2",
		"https://other.example.com/cb",
	}
	for _, r := range registered {
		if !isAllowedRedirectURI(registered, r) {
			t.Errorf("isAllowedRedirectURI: %q should match but did not", r)
		}
	}
}

func TestParseScopes_MultipleScopes(t *testing.T) {
	got := parseScopes("openid email profile")
	want := []string{"openid", "email", "profile"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseScopes = %v, want %v", got, want)
	}
}

func TestParseScopes_SingleScope(t *testing.T) {
	got := parseScopes("openid")
	want := []string{"openid"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseScopes = %v, want %v", got, want)
	}
}

func TestParseScopes_EmptyString(t *testing.T) {
	got := parseScopes("")
	if got != nil {
		t.Errorf("parseScopes(\"\") = %v, want nil", got)
	}
}

func TestParseScopes_ExtraWhitespace(t *testing.T) {
	// strings.Fields handles multiple spaces.
	got := parseScopes("openid  email   profile")
	want := []string{"openid", "email", "profile"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseScopes (extra whitespace) = %v, want %v", got, want)
	}
}

func TestScopesGranted_AllGranted(t *testing.T) {
	granted := []string{"openid", "email", "profile"}
	requested := []string{"openid", "email"}
	if !scopesGranted(granted, requested) {
		t.Error("scopesGranted: all requested scopes are granted, but returned false")
	}
}

func TestScopesGranted_ExactMatch(t *testing.T) {
	granted := []string{"openid", "email"}
	requested := []string{"openid", "email"}
	if !scopesGranted(granted, requested) {
		t.Error("scopesGranted: exact match should return true")
	}
}

func TestScopesGranted_PartialGranted(t *testing.T) {
	granted := []string{"openid"}
	requested := []string{"openid", "email"}
	if scopesGranted(granted, requested) {
		t.Error("scopesGranted: partial grant should return false")
	}
}

func TestScopesGranted_NoneGranted(t *testing.T) {
	granted := []string{"openid"}
	requested := []string{"email", "profile"}
	if scopesGranted(granted, requested) {
		t.Error("scopesGranted: none of the requested scopes are granted, but returned true")
	}
}

func TestScopesGranted_EmptyRequested(t *testing.T) {
	granted := []string{"openid", "email"}
	// Empty requested set is vacuously true.
	if !scopesGranted(granted, nil) {
		t.Error("scopesGranted: empty requested should return true (vacuously)")
	}
}

func TestScopesGranted_EmptyBoth(t *testing.T) {
	if !scopesGranted(nil, nil) {
		t.Error("scopesGranted: both empty should return true")
	}
}

func TestScopesGranted_EmptyGranted(t *testing.T) {
	requested := []string{"openid"}
	if scopesGranted(nil, requested) {
		t.Error("scopesGranted: empty granted with non-empty requested should return false")
	}
}
