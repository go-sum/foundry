package headers

import (
	"testing"
)

func TestParseCookieList(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPairs  []CookiePair
		wantGet    map[string]string
		wantMissed []string
	}{
		{
			name:      "empty",
			input:     "",
			wantPairs: nil,
		},
		{
			name:      "single cookie",
			input:     "session=abc123",
			wantPairs: []CookiePair{{Name: "session", Value: "abc123"}},
			wantGet:   map[string]string{"session": "abc123"},
		},
		{
			name:  "multiple cookies",
			input: "a=1; b=2; c=3",
			wantPairs: []CookiePair{
				{Name: "a", Value: "1"},
				{Name: "b", Value: "2"},
				{Name: "c", Value: "3"},
			},
			wantGet: map[string]string{"a": "1", "b": "2", "c": "3"},
		},
		{
			name:  "duplicate names: get returns first",
			input: "token=first; token=second",
			wantPairs: []CookiePair{
				{Name: "token", Value: "first"},
				{Name: "token", Value: "second"},
			},
			wantGet: map[string]string{"token": "first"},
		},
		{
			name:      "empty value",
			input:     "key=",
			wantPairs: []CookiePair{{Name: "key", Value: ""}},
			wantGet:   map[string]string{"key": ""},
		},
		{
			name:      "__Host- prefixed cookie",
			input:     "__Host-session=abc",
			wantPairs: []CookiePair{{Name: "__Host-session", Value: "abc"}},
			wantGet:   map[string]string{"__Host-session": "abc"},
		},
		{
			name:      "__Secure- prefixed cookie",
			input:     "__Secure-token=xyz",
			wantPairs: []CookiePair{{Name: "__Secure-token", Value: "xyz"}},
			wantGet:   map[string]string{"__Secure-token": "xyz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := ParseCookieList(tt.input)

			all := cl.All()
			if len(all) != len(tt.wantPairs) {
				t.Fatalf("All() returned %d pairs, want %d", len(all), len(tt.wantPairs))
			}
			for i, want := range tt.wantPairs {
				if all[i] != want {
					t.Errorf("pair[%d] = %+v, want %+v", i, all[i], want)
				}
			}

			for name, wantVal := range tt.wantGet {
				got, ok := cl.Get(name)
				if !ok {
					t.Errorf("Get(%q) not found", name)
				} else if got != wantVal {
					t.Errorf("Get(%q) = %q, want %q", name, got, wantVal)
				}
				if !cl.Has(name) {
					t.Errorf("Has(%q) = false, want true", name)
				}
			}
		})
	}
}

func TestCookieListString(t *testing.T) {
	cl := ParseCookieList("a=1; b=2; c=3")
	got := cl.String()
	want := "a=1; b=2; c=3"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
