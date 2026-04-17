package headers

import "testing"

func TestParseIfMatch(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantTags     []string
		wantWildcard bool
		wantErr      bool
	}{
		{
			name:     "single ETag",
			input:    `"abc123"`,
			wantTags: []string{"abc123"},
		},
		{
			name:     "multiple ETags",
			input:    `"abc", "def", "ghi"`,
			wantTags: []string{"abc", "def", "ghi"},
		},
		{
			name:         "wildcard",
			input:        "*",
			wantWildcard: true,
		},
		{
			name:    "weak ETag rejected",
			input:   `W/"abc"`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "unquoted ETag rejected",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIfMatch(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Wildcard != tt.wantWildcard {
				t.Errorf("Wildcard = %v, want %v", got.Wildcard, tt.wantWildcard)
			}
			if len(got.Tags) != len(tt.wantTags) {
				t.Fatalf("got %d tags, want %d", len(got.Tags), len(tt.wantTags))
			}
			for i, want := range tt.wantTags {
				if got.Tags[i] != want {
					t.Errorf("Tags[%d] = %q, want %q", i, got.Tags[i], want)
				}
			}
		})
	}
}

func TestIfMatchMatches(t *testing.T) {
	tests := []struct {
		name   string
		header string
		etag   string
		want   bool
	}{
		{"exact match", `"abc"`, "abc", true},
		{"no match", `"abc"`, "xyz", false},
		{"wildcard matches any", "*", "anything", true},
		{"wildcard does not match empty", "*", "", false},
		{"multiple tags match", `"a", "b", "c"`, "b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := ParseIfMatch(tt.header)
			if err != nil {
				t.Fatalf("ParseIfMatch error: %v", err)
			}
			if got := m.Matches(tt.etag); got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.etag, got, tt.want)
			}
		})
	}
}

func TestIfMatchString(t *testing.T) {
	m := IfMatch{Tags: []string{"abc", "def"}}
	got := m.String()
	want := `"abc", "def"`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	w := IfMatch{Wildcard: true}
	if w.String() != "*" {
		t.Errorf("wildcard String() = %q, want *", w.String())
	}
}
