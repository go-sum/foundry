package headers

import "testing"

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTags  []string
		wantQFirst float64
	}{
		{
			name:       "single language",
			input:      "en-US",
			wantTags:   []string{"en-US"},
			wantQFirst: 1.0,
		},
		{
			name:       "multiple with q",
			input:      "en-US, en;q=0.9, fr;q=0.8",
			wantTags:   []string{"en-US", "en", "fr"},
			wantQFirst: 1.0,
		},
		{
			name:      "empty",
			input:     "",
			wantTags:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			al, err := ParseAcceptLanguage(tt.input)
			if err != nil {
				t.Fatalf("ParseAcceptLanguage(%q) error: %v", tt.input, err)
			}
			if len(al.Languages) != len(tt.wantTags) {
				t.Fatalf("got %d languages, want %d", len(al.Languages), len(tt.wantTags))
			}
			for i, want := range tt.wantTags {
				if al.Languages[i].Tag != want {
					t.Errorf("Languages[%d].Tag = %q, want %q", i, al.Languages[i].Tag, want)
				}
			}
		})
	}
}

func TestAcceptLanguageNegotiate(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		offered []string
		want    string
	}{
		{
			name:    "exact match",
			header:  "en-US",
			offered: []string{"en-US", "fr-FR"},
			want:    "en-US",
		},
		{
			name:    "prefix match en matches en-US",
			header:  "en",
			offered: []string{"en-US", "fr-FR"},
			want:    "en-US",
		},
		{
			name:    "offered prefix matches accept tag",
			header:  "en-US",
			offered: []string{"en", "fr"},
			want:    "en",
		},
		{
			name:    "q ordering: fr preferred",
			header:  "fr;q=1.0, en;q=0.9",
			offered: []string{"en", "fr"},
			want:    "fr",
		},
		{
			name:    "no match",
			header:  "zh-CN",
			offered: []string{"en", "fr"},
			want:    "",
		},
		{
			name:    "empty accept returns first offered",
			header:  "",
			offered: []string{"en-US"},
			want:    "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			al, err := ParseAcceptLanguage(tt.header)
			if err != nil {
				t.Fatalf("ParseAcceptLanguage error: %v", err)
			}
			got := al.Negotiate(tt.offered...)
			if got != tt.want {
				t.Errorf("Negotiate() = %q, want %q", got, tt.want)
			}
		})
	}
}
