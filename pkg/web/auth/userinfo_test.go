package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchUserinfo_Success(t *testing.T) {
	want := UserinfoClaims{
		Sub:           "user123",
		Email:         "user@example.com",
		EmailVerified: true,
		Name:          "Test User",
		Picture:       "https://example.com/pic.jpg",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := FetchUserinfo(context.Background(), srv.Client(), srv.URL, "mytoken")
	if err != nil {
		t.Fatalf("FetchUserinfo error: %v", err)
	}
	if got != want {
		t.Errorf("claims = %+v, want %+v", got, want)
	}
}

func TestFetchUserinfo_SendsBearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(UserinfoClaims{Sub: "u1"})
	}))
	defer srv.Close()

	if _, err := FetchUserinfo(context.Background(), srv.Client(), srv.URL, "secrettoken"); err != nil {
		t.Fatalf("FetchUserinfo error: %v", err)
	}
	if gotAuth != "Bearer secrettoken" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer secrettoken")
	}
}

func TestFetchUserinfo_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := FetchUserinfo(context.Background(), srv.Client(), srv.URL, "tok")
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

func TestFetchUserinfo_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer srv.Close()

	_, err := FetchUserinfo(context.Background(), srv.Client(), srv.URL, "tok")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
