package headers

import (
	"testing"
)

func TestParseAccept(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTypes  []string // in order (sorted by q desc)
		wantQFirst float64
	}{
		{
			name:       "single type",
			input:      "text/html",
			wantTypes:  []string{"text/html"},
			wantQFirst: 1.0,
		},
		{
			name:       "multiple types with q",
			input:      "text/html, application/json;q=0.9, */*;q=0.8",
			wantTypes:  []string{"text/html", "application/json", "*/*"},
			wantQFirst: 1.0,
		},
		{
			name:      "empty input",
			input:     "",
			wantTypes: nil,
		},
		{
			name:       "q=0 excluded from negotiation but present",
			input:      "text/html;q=0, application/json;q=1.0",
			wantTypes:  []string{"application/json", "text/html"},
			wantQFirst: 1.0,
		},
		{
			name:       "wildcard only",
			input:      "*/*",
			wantTypes:  []string{"*/*"},
			wantQFirst: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := ParseAccept(tt.input)
			if err != nil {
				t.Fatalf("ParseAccept(%q) unexpected error: %v", tt.input, err)
			}
			if len(a.MediaTypes) != len(tt.wantTypes) {
				t.Fatalf("got %d media types, want %d", len(a.MediaTypes), len(tt.wantTypes))
			}
			for i, want := range tt.wantTypes {
				got := a.MediaTypes[i].Type + "/" + a.MediaTypes[i].Sub
				if got != want {
					t.Errorf("MediaTypes[%d] = %q, want %q", i, got, want)
				}
			}
			if len(a.MediaTypes) > 0 && a.MediaTypes[0].Q != tt.wantQFirst {
				t.Errorf("first Q = %v, want %v", a.MediaTypes[0].Q, tt.wantQFirst)
			}
		})
	}
}

func TestAcceptNegotiate(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		offered []string
		want    string
	}{
		{
			name:    "exact match",
			header:  "text/html",
			offered: []string{"text/html", "application/json"},
			want:    "text/html",
		},
		{
			name:    "wildcard type/* matches",
			header:  "text/*",
			offered: []string{"application/json", "text/html"},
			want:    "text/html",
		},
		{
			name:    "wildcard */* matches any",
			header:  "*/*",
			offered: []string{"application/json"},
			want:    "application/json",
		},
		{
			name:    "q=0 excludes type",
			header:  "text/html;q=0, application/json;q=1.0",
			offered: []string{"text/html"},
			want:    "",
		},
		{
			name:    "exact beats wildcard",
			header:  "text/html;q=1.0, text/*;q=0.9",
			offered: []string{"text/plain", "text/html"},
			want:    "text/html",
		},
		{
			name:    "empty accept accepts all",
			header:  "",
			offered: []string{"application/json"},
			want:    "application/json",
		},
		{
			name:    "no offered types",
			header:  "text/html",
			offered: []string{},
			want:    "",
		},
		{
			name:    "no match",
			header:  "text/html",
			offered: []string{"application/json"},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := ParseAccept(tt.header)
			if err != nil {
				t.Fatalf("ParseAccept error: %v", err)
			}
			got := a.Negotiate(tt.offered...)
			if got != tt.want {
				t.Errorf("Negotiate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAcceptString(t *testing.T) {
	input := "text/html, application/json;q=0.9, */*;q=0.8"
	a, err := ParseAccept(input)
	if err != nil {
		t.Fatal(err)
	}
	got := a.String()
	// Re-parse and verify content is equivalent
	a2, err := ParseAccept(got)
	if err != nil {
		t.Fatalf("round-trip parse error: %v", err)
	}
	if len(a.MediaTypes) != len(a2.MediaTypes) {
		t.Errorf("round-trip lost media types: got %d, want %d", len(a2.MediaTypes), len(a.MediaTypes))
	}
}
