package web

import (
	"net/http"
	"sort"
	"strings"
	"testing"
)

func TestHeaders_Get(t *testing.T) {
	tests := []struct {
		name  string
		setup func() Headers
		key   string
		want  string
	}{
		{
			name:  "returns first value for existing key",
			setup: func() Headers { h := NewHeaders(); h.Set("Content-Type", "text/html"); return h },
			key:   "Content-Type",
			want:  "text/html",
		},
		{
			name:  "case insensitive lookup",
			setup: func() Headers { h := NewHeaders(); h.Set("content-type", "text/html"); return h },
			key:   "CONTENT-TYPE",
			want:  "text/html",
		},
		{
			name:  "returns empty string for missing key",
			setup: func() Headers { return NewHeaders() },
			key:   "X-Missing",
			want:  "",
		},
		{
			name: "returns first value when multiple appended",
			setup: func() Headers {
				h := NewHeaders()
				h.Append("Accept", "text/html")
				h.Append("Accept", "application/json")
				return h
			},
			key:  "Accept",
			want: "text/html",
		},
		{
			name:  "zero value Headers returns empty string",
			setup: func() Headers { return Headers{} },
			key:   "Any",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setup()
			got := h.Get(tt.key)
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestHeaders_Set(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Headers
		setKey   string
		setValue string
		queryKey string
		wantVals []string
	}{
		{
			name:   "sets single value",
			setup:  func() *Headers { h := NewHeaders(); return &h },
			setKey: "X-Custom", setValue: "val1",
			queryKey: "x-custom", wantVals: []string{"val1"},
		},
		{
			name: "replaces existing values",
			setup: func() *Headers {
				h := NewHeaders()
				h.Append("X-Custom", "old1")
				h.Append("X-Custom", "old2")
				return &h
			},
			setKey: "X-Custom", setValue: "new",
			queryKey: "x-custom", wantVals: []string{"new"},
		},
		{
			name:   "zero value Headers does not panic",
			setup:  func() *Headers { h := Headers{}; return &h },
			setKey: "Key", setValue: "val",
			queryKey: "key", wantVals: []string{"val"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setup()
			h.Set(tt.setKey, tt.setValue)
			got := h.Values(tt.queryKey)
			if !sliceEqual(got, tt.wantVals) {
				t.Errorf("after Set(%q, %q): Values(%q) = %v, want %v", tt.setKey, tt.setValue, tt.queryKey, got, tt.wantVals)
			}
		})
	}
}

func TestHeaders_Append(t *testing.T) {
	t.Run("appends without replacing", func(t *testing.T) {
		h := NewHeaders()
		h.Append("Accept", "text/html")
		h.Append("Accept", "application/json")
		got := h.Values("accept")
		want := []string{"text/html", "application/json"}
		if !sliceEqual(got, want) {
			t.Errorf("Values = %v, want %v", got, want)
		}
	})

	t.Run("zero value Headers does not panic", func(t *testing.T) {
		h := Headers{}
		h.Append("Key", "val")
		got := h.Get("key")
		if got != "val" {
			t.Errorf("Get = %q, want %q", got, "val")
		}
	})
}

func TestHeaders_Delete(t *testing.T) {
	t.Run("removes existing header", func(t *testing.T) {
		h := NewHeaders()
		h.Set("X-Remove", "val")
		h.Delete("X-Remove")
		if h.Has("x-remove") {
			t.Error("Has returned true after Delete")
		}
	})

	t.Run("delete on missing key does not panic", func(t *testing.T) {
		h := NewHeaders()
		h.Delete("X-Missing") // should not panic
	})

	t.Run("delete on zero value Headers does not panic", func(t *testing.T) {
		h := Headers{}
		h.Delete("X-Missing") // should not panic
	})
}

func TestHeaders_Has(t *testing.T) {
	tests := []struct {
		name  string
		setup func() Headers
		key   string
		want  bool
	}{
		{
			name:  "returns true for existing key",
			setup: func() Headers { h := NewHeaders(); h.Set("X-Exists", "v"); return h },
			key:   "x-exists",
			want:  true,
		},
		{
			name:  "returns false for missing key",
			setup: func() Headers { return NewHeaders() },
			key:   "X-Missing",
			want:  false,
		},
		{
			name:  "zero value Headers returns false",
			setup: func() Headers { return Headers{} },
			key:   "Any",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setup()
			if got := h.Has(tt.key); got != tt.want {
				t.Errorf("Has(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestHeaders_Values(t *testing.T) {
	t.Run("returns all values for key", func(t *testing.T) {
		h := NewHeaders()
		h.Append("Accept", "a")
		h.Append("Accept", "b")
		got := h.Values("accept")
		want := []string{"a", "b"}
		if !sliceEqual(got, want) {
			t.Errorf("Values = %v, want %v", got, want)
		}
	})

	t.Run("returns nil for missing key", func(t *testing.T) {
		h := NewHeaders()
		got := h.Values("missing")
		if got != nil {
			t.Errorf("Values = %v, want nil", got)
		}
	})

	t.Run("zero value Headers returns nil", func(t *testing.T) {
		h := Headers{}
		got := h.Values("any")
		if got != nil {
			t.Errorf("Values = %v, want nil", got)
		}
	})

	t.Run("returned slice is a copy", func(t *testing.T) {
		h := NewHeaders()
		h.Append("Accept", "a")
		h.Append("Accept", "b")

		got := h.Values("accept")
		got[0] = "mutated"

		if h.Get("Accept") != "a" {
			t.Fatalf("mutating Values() result affected original headers")
		}
	})
}

func TestHeaders_Entries(t *testing.T) {
	t.Run("returns copy of all entries", func(t *testing.T) {
		h := NewHeaders()
		h.Set("a", "1")
		h.Append("b", "2")
		h.Append("b", "3")
		entries := h.Entries()
		if len(entries) != 2 {
			t.Fatalf("len(entries) = %d, want 2", len(entries))
		}
		if !sliceEqual(entries["a"], []string{"1"}) {
			t.Errorf("entries[a] = %v, want [1]", entries["a"])
		}
		if !sliceEqual(entries["b"], []string{"2", "3"}) {
			t.Errorf("entries[b] = %v, want [2 3]", entries["b"])
		}
	})

	t.Run("mutating returned map does not affect original", func(t *testing.T) {
		h := NewHeaders()
		h.Set("key", "val")
		entries := h.Entries()
		entries["key"][0] = "mutated"
		if h.Get("key") != "val" {
			t.Error("mutating Entries return value affected original")
		}
	})

	t.Run("zero value Headers returns nil", func(t *testing.T) {
		h := Headers{}
		if entries := h.Entries(); entries != nil {
			t.Errorf("Entries = %v, want nil", entries)
		}
	})
}

func TestHeaders_Clone(t *testing.T) {
	t.Run("clone is independent from original", func(t *testing.T) {
		h := NewHeaders()
		h.Set("X-Original", "val")
		c := h.Clone()

		// mutate clone
		c.Set("X-Original", "changed")
		c.Set("X-New", "added")

		if h.Get("X-Original") != "val" {
			t.Error("mutating clone affected original value")
		}
		if h.Has("X-New") {
			t.Error("adding to clone affected original")
		}
	})

	t.Run("clone of zero value is usable", func(t *testing.T) {
		h := Headers{}
		c := h.Clone()
		c.Set("Key", "val") // should not panic
		if c.Get("Key") != "val" {
			t.Errorf("Get on clone = %q, want %q", c.Get("Key"), "val")
		}
	})
}

func TestHeaders_ForEach(t *testing.T) {
	t.Run("iterates all entries", func(t *testing.T) {
		h := NewHeaders()
		h.Set("a", "1")
		h.Set("b", "2")
		visited := make(map[string][]string)
		h.ForEach(func(name string, values []string) {
			visited[name] = values
		})
		if len(visited) != 2 {
			t.Fatalf("visited %d entries, want 2", len(visited))
		}
		if !sliceEqual(visited["a"], []string{"1"}) {
			t.Errorf("visited[a] = %v, want [1]", visited["a"])
		}
		if !sliceEqual(visited["b"], []string{"2"}) {
			t.Errorf("visited[b] = %v, want [2]", visited["b"])
		}
	})

	t.Run("zero value Headers does not panic", func(t *testing.T) {
		h := Headers{}
		h.ForEach(func(name string, values []string) {
			t.Error("should not be called on empty headers")
		})
	})
}

func TestNewHeaders(t *testing.T) {
	h := NewHeaders()
	// Should be usable immediately
	h.Set("K", "V")
	if got := h.Get("k"); got != "V" {
		t.Errorf("Get = %q, want %q", got, "V")
	}
}

func TestP0_02_Headers_CRLFRejection(t *testing.T) {
	corpus := crlfCorpus()
	for _, payload := range corpus {
		t.Run("value="+payload, func(t *testing.T) {
			h := NewHeaders()
			h.Set("X-Test", payload)
			got := h.Get("X-Test")
			assertNoCRLF(t, "Set then Get", got)
		})
		t.Run("append="+payload, func(t *testing.T) {
			h := NewHeaders()
			h.Append("X-Test", payload)
			got := h.Get("X-Test")
			assertNoCRLF(t, "Append then Get", got)
		})
	}
}

func TestHeaders_GetSetCookie(t *testing.T) {
	h := NewHeaders()
	h.Append("Set-Cookie", "session=abc; Path=/; HttpOnly")
	h.Append("Set-Cookie", "lang=en; Path=/")

	got := h.GetSetCookie()
	if len(got) != 2 {
		t.Fatalf("GetSetCookie() len = %d, want 2", len(got))
	}
	if got[0] != "session=abc; Path=/; HttpOnly" {
		t.Errorf("GetSetCookie()[0] = %q, want %q", got[0], "session=abc; Path=/; HttpOnly")
	}
	if got[1] != "lang=en; Path=/" {
		t.Errorf("GetSetCookie()[1] = %q, want %q", got[1], "lang=en; Path=/")
	}
}

func TestIsForbiddenResponseHeader(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Transfer-Encoding", true},
		{"Connection", true},
		{"Keep-Alive", true},
		{"Upgrade", true},
		{"Trailer", true},
		{"Content-Type", false},
		{"X-Custom", false},
		{"Content-Length", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsForbiddenResponseHeader(tt.name)
			if got != tt.want {
				t.Errorf("IsForbiddenResponseHeader(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsForbiddenResponseHeaderForStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{"Upgrade", 200, true},
		{"Upgrade", 101, false},
		{"Connection", 101, false},
		{"Transfer-Encoding", 101, true},
		{"Content-Type", 200, false},
	}
	for _, tt := range tests {
		t.Run(tt.name+"_"+http.StatusText(tt.status), func(t *testing.T) {
			got := IsForbiddenResponseHeaderForStatus(tt.name, tt.status)
			if got != tt.want {
				t.Errorf("IsForbiddenResponseHeaderForStatus(%q, %d) = %v, want %v", tt.name, tt.status, got, tt.want)
			}
		})
	}
}

// crlfCorpus returns a set of payloads that attempt CRLF header injection.
func crlfCorpus() []string {
	return []string{
		"innocent\r\nSet-Cookie: evil=1",
		"innocent\rSet-Cookie: evil=1",
		"innocent\nSet-Cookie: evil=1",
		"innocent\r\n\r\n<script>alert(1)</script>",
		"\r\n",
		"\r",
		"\n",
		"a\x0db",
		"a\x0ab",
	}
}

// assertNoCRLF asserts that s contains no carriage return or line feed bytes.
func assertNoCRLF(t *testing.T, label, s string) {
	t.Helper()
	if strings.ContainsAny(s, "\r\n") {
		t.Errorf("%s: contains CR or LF: %q", label, s)
	}
}

// sliceEqual compares two string slices for equality.
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sortedKeys returns sorted keys from a map for deterministic comparison.
func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
