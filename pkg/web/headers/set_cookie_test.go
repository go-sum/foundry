package headers

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestParseSetCookie(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	refTimeStr := refTime.Format(http.TimeFormat)

	tests := []struct {
		name    string
		input   string
		want    SetCookie
		wantErr bool
	}{
		{
			name:  "simple cookie",
			input: "session=abc123",
			want:  SetCookie{Name: "session", Value: "abc123"},
		},
		{
			name:  "full cookie",
			input: "id=xyz; Domain=example.com; Path=/; HttpOnly; Secure; SameSite=Strict",
			want: SetCookie{
				Name:     "id",
				Value:    "xyz",
				Domain:   "example.com",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: "Strict",
			},
		},
		{
			name:  "canonicalizes lowercase SameSite and Priority",
			input: "id=xyz; SameSite=none; Priority=medium",
			want: SetCookie{
				Name:     "id",
				Value:    "xyz",
				SameSite: "None",
				Priority: "Medium",
			},
		},
		{
			name:  "max-age",
			input: "token=abc; Max-Age=3600",
			want:  SetCookie{Name: "token", Value: "abc", MaxAge: intPtr(3600)},
		},
		{
			name:  "max-age=0 (delete)",
			input: "token=; Max-Age=0",
			want:  SetCookie{Name: "token", Value: "", MaxAge: intPtr(0)},
		},
		{
			name:  "expires",
			input: "key=val; Expires=" + refTimeStr,
			want:  SetCookie{Name: "key", Value: "val", Expires: refTime},
		},
		{
			name:  "partitioned forces secure",
			input: "key=val; Partitioned",
			want:  SetCookie{Name: "key", Value: "val", Partitioned: true},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSetCookie(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Value != tt.want.Value {
				t.Errorf("Value = %q, want %q", got.Value, tt.want.Value)
			}
			if got.Domain != tt.want.Domain {
				t.Errorf("Domain = %q, want %q", got.Domain, tt.want.Domain)
			}
			if got.Path != tt.want.Path {
				t.Errorf("Path = %q, want %q", got.Path, tt.want.Path)
			}
			if got.HttpOnly != tt.want.HttpOnly {
				t.Errorf("HttpOnly = %v, want %v", got.HttpOnly, tt.want.HttpOnly)
			}
			if got.Secure != tt.want.Secure {
				t.Errorf("Secure = %v, want %v", got.Secure, tt.want.Secure)
			}
			if got.SameSite != tt.want.SameSite {
				t.Errorf("SameSite = %q, want %q", got.SameSite, tt.want.SameSite)
			}
			if got.Priority != tt.want.Priority {
				t.Errorf("Priority = %q, want %q", got.Priority, tt.want.Priority)
			}
			if got.Partitioned != tt.want.Partitioned {
				t.Errorf("Partitioned = %v, want %v", got.Partitioned, tt.want.Partitioned)
			}
			compareIntPtr(t, "MaxAge", got.MaxAge, tt.want.MaxAge)
			if !got.Expires.Equal(tt.want.Expires) {
				t.Errorf("Expires = %v, want %v", got.Expires, tt.want.Expires)
			}
		})
	}
}

func TestSetCookieString(t *testing.T) {
	t.Run("samesite defaults to lax", func(t *testing.T) {
		sc := SetCookie{Name: "a", Value: "b"}
		got := sc.String()
		if !strings.Contains(got, "SameSite=Lax") {
			t.Errorf("expected SameSite=Lax in %q", got)
		}
	})

	t.Run("partitioned forces secure", func(t *testing.T) {
		sc := SetCookie{Name: "a", Value: "b", Partitioned: true}
		got := sc.String()
		if !strings.Contains(got, "; Secure") {
			t.Errorf("expected Secure in %q due to Partitioned", got)
		}
	})

	t.Run("max-age=0 present", func(t *testing.T) {
		n := 0
		sc := SetCookie{Name: "a", Value: "b", MaxAge: &n}
		got := sc.String()
		if !strings.Contains(got, "Max-Age=0") {
			t.Errorf("expected Max-Age=0 in %q", got)
		}
	})

	t.Run("expires formatted as http date", func(t *testing.T) {
		refTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		sc := SetCookie{Name: "a", Value: "b", Expires: refTime}
		got := sc.String()
		want := refTime.Format(http.TimeFormat)
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in %q", want, got)
		}
	})
}
