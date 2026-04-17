package headers

import "testing"

func TestParseAcceptEncoding(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTokens []string
	}{
		{
			name:       "single encoding",
			input:      "gzip",
			wantTokens: []string{"gzip"},
		},
		{
			name:       "multiple with q values",
			input:      "br;q=1.0, gzip;q=0.9, deflate;q=0.8",
			wantTokens: []string{"br", "gzip", "deflate"},
		},
		{
			name:       "empty input",
			input:      "",
			wantTokens: nil,
		},
		{
			name:       "identity with q=0",
			input:      "identity;q=0",
			wantTokens: []string{"identity"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae, err := ParseAcceptEncoding(tt.input)
			if err != nil {
				t.Fatalf("ParseAcceptEncoding(%q) error: %v", tt.input, err)
			}
			if len(ae.Encodings) != len(tt.wantTokens) {
				t.Fatalf("got %d encodings, want %d", len(ae.Encodings), len(tt.wantTokens))
			}
			for i, want := range tt.wantTokens {
				if ae.Encodings[i].Token != want {
					t.Errorf("Encodings[%d].Token = %q, want %q", i, ae.Encodings[i].Token, want)
				}
			}
		})
	}
}

func TestAcceptEncodingNegotiate(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		offered []string
		want    string
	}{
		{
			name:    "prefer br over gzip",
			header:  "br;q=1.0, gzip;q=0.9",
			offered: []string{"br", "gzip"},
			want:    "br",
		},
		{
			name:    "identity q=0 rejected",
			header:  "identity;q=0",
			offered: []string{"identity"},
			want:    "",
		},
		{
			name:    "wildcard q=0 rejects all",
			header:  "*;q=0",
			offered: []string{"gzip", "br", "identity"},
			want:    "",
		},
		{
			name:    "empty header accepts identity",
			header:  "",
			offered: []string{"identity"},
			want:    "identity",
		},
		{
			name:    "empty header accepts first offered",
			header:  "",
			offered: []string{"gzip", "br"},
			want:    "gzip",
		},
		{
			name:    "identity implicitly acceptable when not listed",
			header:  "gzip",
			offered: []string{"identity"},
			want:    "identity",
		},
		{
			name:    "gzip not listed, not via wildcard",
			header:  "br",
			offered: []string{"gzip"},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae, err := ParseAcceptEncoding(tt.header)
			if err != nil {
				t.Fatalf("ParseAcceptEncoding error: %v", err)
			}
			got := ae.Negotiate(tt.offered...)
			if got != tt.want {
				t.Errorf("Negotiate() = %q, want %q", got, tt.want)
			}
		})
	}
}
