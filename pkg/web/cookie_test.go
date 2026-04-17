package web

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseCookies(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   []Cookie
	}{
		{
			name:   "two cookies",
			header: "name1=val1; name2=val2",
			want: []Cookie{
				{Name: "name1", Value: "val1"},
				{Name: "name2", Value: "val2"},
			},
		},
		{
			name:   "empty string",
			header: "",
			want:   nil,
		},
		{
			name:   "single cookie no spaces",
			header: "session=abc123",
			want:   []Cookie{{Name: "session", Value: "abc123"}},
		},
		{
			name:   "extra whitespace",
			header: " foo=bar ;  baz=qux ",
			want: []Cookie{
				{Name: "foo", Value: "bar"},
				{Name: "baz", Value: "qux"},
			},
		},
		{
			name:   "empty pair is skipped",
			header: "a=1;; b=2",
			want: []Cookie{
				{Name: "a", Value: "1"},
				{Name: "b", Value: "2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCookies(tt.header)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("cookie[%d].Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}
				if got[i].Value != tt.want[i].Value {
					t.Errorf("cookie[%d].Value = %q, want %q", i, got[i].Value, tt.want[i].Value)
				}
			}
		})
	}
}

func TestCookieString(t *testing.T) {
	tests := []struct {
		name   string
		cookie Cookie
		want   string
	}{
		{
			name: "all attributes set",
			cookie: Cookie{
				Name:     "session",
				Value:    "abc123",
				Path:     "/",
				Domain:   "example.com",
				MaxAge:   3600,
				Secure:   true,
				HTTPOnly: true,
				SameSite: "Strict",
			},
			want: "session=abc123; Path=/; Domain=example.com; Max-Age=3600; Secure; HttpOnly; SameSite=Strict",
		},
		{
			// SameSite="" emits "SameSite=Lax" by default.
			name: "minimal attributes",
			cookie: Cookie{
				Name:  "token",
				Value: "xyz",
			},
			want: "token=xyz; SameSite=Lax",
		},
		{
			name: "negative MaxAge outputs Max-Age=0",
			cookie: Cookie{
				Name:   "expired",
				Value:  "old",
				MaxAge: -1,
			},
			want: "expired=old; Max-Age=0; SameSite=Lax",
		},
		{
			name: "path and SameSite Lax explicit",
			cookie: Cookie{
				Name:     "pref",
				Value:    "dark",
				Path:     "/app",
				SameSite: "Lax",
			},
			want: "pref=dark; Path=/app; SameSite=Lax",
		},
		{
			name: "unknown SameSite is preserved verbatim",
			cookie: Cookie{
				Name:     "mode",
				Value:    "custom",
				SameSite: "Experimental",
			},
			want: "mode=custom; SameSite=Experimental",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cookie.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetCookie(t *testing.T) {
	resp := Respond(200)
	cookie := Cookie{Name: "sid", Value: "abc", Path: "/", SameSite: "Lax"}
	SetCookie(&resp, cookie)

	values := resp.Headers.Values("Set-Cookie")
	if len(values) != 1 {
		t.Fatalf("Set-Cookie header count = %d, want 1", len(values))
	}
	want := "sid=abc; Path=/; SameSite=Lax"
	if values[0] != want {
		t.Errorf("Set-Cookie = %q, want %q", values[0], want)
	}

	// Append a second cookie.
	cookie2 := Cookie{Name: "lang", Value: "en", SameSite: "Lax"}
	SetCookie(&resp, cookie2)
	values = resp.Headers.Values("Set-Cookie")
	if len(values) != 2 {
		t.Fatalf("Set-Cookie header count = %d, want 2", len(values))
	}
	if values[1] != "lang=en; SameSite=Lax" {
		t.Errorf("Set-Cookie[1] = %q, want %q", values[1], "lang=en; SameSite=Lax")
	}
}

func TestGetCookie(t *testing.T) {
	t.Run("finds named cookie", func(t *testing.T) {
		req := NewRequest("GET", &url.URL{Path: "/"})
		req.Headers.Set("Cookie", "alpha=1; beta=2; gamma=3")

		got, ok := GetCookie(req, "beta")
		if !ok {
			t.Fatal("expected ok = true")
		}
		if got.Name != "beta" {
			t.Errorf("Name = %q, want %q", got.Name, "beta")
		}
		if got.Value != "2" {
			t.Errorf("Value = %q, want %q", got.Value, "2")
		}
	})

	t.Run("returns false for missing name", func(t *testing.T) {
		req := NewRequest("GET", &url.URL{Path: "/"})
		req.Headers.Set("Cookie", "alpha=1")

		_, ok := GetCookie(req, "missing")
		if ok {
			t.Error("expected ok = false for missing cookie")
		}
	})

	t.Run("returns false for empty cookie header", func(t *testing.T) {
		req := NewRequest("GET", &url.URL{Path: "/"})

		_, ok := GetCookie(req, "anything")
		if ok {
			t.Error("expected ok = false for empty cookie header")
		}
	})
}

func TestCookie_String_WithExpires(t *testing.T) {
	expires := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	c := Cookie{
		Name:     "session",
		Value:    "abc",
		Expires:  expires,
		MaxAge:   3600,
		SameSite: "Strict",
	}
	got := c.String()
	want := "session=abc; Max-Age=3600; Expires=Thu, 15 Jan 2026 12:00:00 GMT; SameSite=Strict"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestCookie_String_QuotesUnsafeValue(t *testing.T) {
	c := Cookie{Name: "pref", Value: "a;b", SameSite: "Lax"}
	got := c.String()
	// Value contains semicolon — must be quoted.
	if !strings.HasPrefix(got, `pref="a;b"`) {
		t.Errorf("String() = %q, want prefix pref=\"a;b\"", got)
	}
}

func TestCookie_String_SafeValue_NotQuoted(t *testing.T) {
	c := Cookie{Name: "token", Value: "abc123", SameSite: "Lax"}
	got := c.String()
	if !strings.HasPrefix(got, "token=abc123") {
		t.Errorf("String() = %q, want prefix token=abc123", got)
	}
}

// TestP0_02_Headers_CRLFRejection_Cookie verifies CRLF is stripped from all cookie fields.
func TestP0_02_Headers_CRLFRejection_Cookie(t *testing.T) {
	corpus := cookieCRLFCorpus()
	for _, payload := range corpus {
		t.Run("value="+payload, func(t *testing.T) {
			c := Cookie{Name: "test", Value: payload, SameSite: "Lax"}
			got := c.String()
			assertCookieNoCRLF(t, "cookie Value in String()", got)
		})
		t.Run("path="+payload, func(t *testing.T) {
			c := Cookie{Name: "test", Value: "v", Path: payload, SameSite: "Lax"}
			got := c.String()
			assertCookieNoCRLF(t, "cookie Path in String()", got)
		})
		t.Run("domain="+payload, func(t *testing.T) {
			c := Cookie{Name: "test", Value: "v", Domain: payload, SameSite: "Lax"}
			got := c.String()
			assertCookieNoCRLF(t, "cookie Domain in String()", got)
		})
	}
}

func cookieCRLFCorpus() []string {
	return []string{
		"innocent\r\nSet-Cookie: evil=1",
		"innocent\rSet-Cookie: evil=1",
		"innocent\nSet-Cookie: evil=1",
		"\r\n",
		"\r",
		"\n",
		"a\x0db",
		"a\x0ab",
	}
}

func assertCookieNoCRLF(t *testing.T, label, s string) {
	t.Helper()
	if strings.ContainsAny(s, "\r\n") {
		t.Errorf("%s: contains CR or LF: %q", label, s)
	}
}

func TestCookie_HostPrefix(t *testing.T) {
	t.Run("valid __Host- cookie", func(t *testing.T) {
		c := Cookie{
			Name:     "__Host-session",
			Value:    "abc",
			Secure:   true,
			Path:     "/",
			SameSite: "Lax",
		}
		if err := c.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
		s := c.String()
		if !strings.Contains(s, "; Secure") {
			t.Errorf("String() missing Secure: %q", s)
		}
		if !strings.Contains(s, "; Path=/") {
			t.Errorf("String() missing Path=/: %q", s)
		}
	})

	t.Run("__Host- without Secure fails Validate", func(t *testing.T) {
		c := Cookie{Name: "__Host-session", Value: "abc", Path: "/"}
		if err := c.Validate(); err == nil {
			t.Error("Validate() = nil, want error for __Host- without Secure")
		}
	})

	t.Run("__Host- without Path=/ fails Validate", func(t *testing.T) {
		c := Cookie{Name: "__Host-session", Value: "abc", Secure: true, Path: "/sub"}
		if err := c.Validate(); err == nil {
			t.Error("Validate() = nil, want error for __Host- without Path=/")
		}
	})

	t.Run("__Host- with Domain fails Validate", func(t *testing.T) {
		c := Cookie{Name: "__Host-session", Value: "abc", Secure: true, Path: "/", Domain: "example.com"}
		if err := c.Validate(); err == nil {
			t.Error("Validate() = nil, want error for __Host- with Domain set")
		}
	})

	t.Run("__Host- String forces Secure and Path=/", func(t *testing.T) {
		c := Cookie{
			Name:  "__Host-x",
			Value: "1",
			// Not setting Secure or Path — String() should enforce them.
		}
		s := c.String()
		if !strings.Contains(s, "; Secure") {
			t.Errorf("String() missing Secure for __Host-: %q", s)
		}
		if !strings.Contains(s, "; Path=/") {
			t.Errorf("String() missing Path=/ for __Host-: %q", s)
		}
	})
}

func TestCookie_SecurePrefix(t *testing.T) {
	t.Run("__Secure- without Secure fails Validate", func(t *testing.T) {
		c := Cookie{Name: "__Secure-token", Value: "abc"}
		if err := c.Validate(); err == nil {
			t.Error("Validate() = nil, want error for __Secure- without Secure")
		}
	})

	t.Run("__Secure- with Secure passes Validate", func(t *testing.T) {
		c := Cookie{Name: "__Secure-token", Value: "abc", Secure: true}
		if err := c.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("__Secure- String forces Secure", func(t *testing.T) {
		c := Cookie{Name: "__Secure-x", Value: "1"}
		s := c.String()
		if !strings.Contains(s, "; Secure") {
			t.Errorf("String() missing Secure for __Secure-: %q", s)
		}
	})
}

func TestCookie_PartitionedForcesSecure(t *testing.T) {
	c := Cookie{
		Name:        "part",
		Value:       "v",
		Partitioned: true,
		SameSite:    "None",
	}
	s := c.String()
	if !strings.Contains(s, "; Secure") {
		t.Errorf("String() missing Secure for Partitioned cookie: %q", s)
	}
	if !strings.Contains(s, "; Partitioned") {
		t.Errorf("String() missing Partitioned attribute: %q", s)
	}
}

func TestCookie_SameSiteLaxDefault(t *testing.T) {
	c := Cookie{Name: "x", Value: "y"}
	s := c.String()
	if !strings.Contains(s, "; SameSite=Lax") {
		t.Errorf("String() with empty SameSite should emit SameSite=Lax, got: %q", s)
	}
}

func TestCookie_Priority(t *testing.T) {
	tests := []struct {
		priority string
		want     string
	}{
		{"High", "; Priority=High"},
		{"Medium", "; Priority=Medium"},
		{"Low", "; Priority=Low"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run("priority="+tt.priority, func(t *testing.T) {
			c := Cookie{Name: "x", Value: "y", SameSite: "Lax", Priority: tt.priority}
			s := c.String()
			if tt.want == "" {
				if strings.Contains(s, "Priority") {
					t.Errorf("String() should not contain Priority when empty: %q", s)
				}
			} else {
				if !strings.Contains(s, tt.want) {
					t.Errorf("String() = %q, want to contain %q", s, tt.want)
				}
			}
		})
	}
}

func TestCookie_MaxAge(t *testing.T) {
	tests := []struct {
		name     string
		maxAge   int
		wantFrag string
		absent   string
	}{
		{"positive sets Max-Age=N", 3600, "; Max-Age=3600", ""},
		{"negative sets Max-Age=0", -1, "; Max-Age=0", ""},
		{"zero omits Max-Age", 0, "", "Max-Age"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Cookie{Name: "x", Value: "y", MaxAge: tt.maxAge, SameSite: "Lax"}
			s := c.String()
			if tt.wantFrag != "" && !strings.Contains(s, tt.wantFrag) {
				t.Errorf("String() = %q, want to contain %q", s, tt.wantFrag)
			}
			if tt.absent != "" && strings.Contains(s, tt.absent) {
				t.Errorf("String() = %q, should not contain %q", s, tt.absent)
			}
		})
	}
}
