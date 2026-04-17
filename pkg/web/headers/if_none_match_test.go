package headers

import "testing"

func TestParseIfNoneMatch(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantStrong   []string
		wantWeak     []string
		wantWildcard bool
		wantErr      bool
	}{
		{
			name:       "single strong ETag",
			input:      `"abc"`,
			wantStrong: []string{"abc"},
		},
		{
			name:     "single weak ETag",
			input:    `W/"abc"`,
			wantWeak: []string{"abc"},
		},
		{
			name:       "mixed strong and weak",
			input:      `"abc", W/"def"`,
			wantStrong: []string{"abc"},
			wantWeak:   []string{"def"},
		},
		{
			name:         "wildcard",
			input:        "*",
			wantWildcard: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIfNoneMatch(tt.input)
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
			if len(got.Tags) != len(tt.wantStrong) {
				t.Errorf("Tags len = %d, want %d", len(got.Tags), len(tt.wantStrong))
			}
			for i, want := range tt.wantStrong {
				if got.Tags[i] != want {
					t.Errorf("Tags[%d] = %q, want %q", i, got.Tags[i], want)
				}
			}
			if len(got.Weak) != len(tt.wantWeak) {
				t.Errorf("Weak len = %d, want %d", len(got.Weak), len(tt.wantWeak))
			}
			for i, want := range tt.wantWeak {
				if got.Weak[i] != want {
					t.Errorf("Weak[%d] = %q, want %q", i, got.Weak[i], want)
				}
			}
		})
	}
}

func TestIfNoneMatchMatches(t *testing.T) {
	tests := []struct {
		name   string
		header string
		etag   string
		weak   bool
		want   bool
	}{
		{"strong match", `"abc"`, "abc", false, true},
		{"strong no match", `"abc"`, "xyz", false, false},
		{"weak tag matches with weak=true", `W/"abc"`, "abc", true, true},
		{"weak tag does NOT match with weak=false", `W/"abc"`, "abc", false, false},
		{"wildcard matches", "*", "any", false, true},
		{"strong in mixed list", `"a", W/"b"`, "a", false, true},
		{"weak in mixed list with weak=true", `"a", W/"b"`, "b", true, true},
		{"weak in mixed list with weak=false", `"a", W/"b"`, "b", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := ParseIfNoneMatch(tt.header)
			if err != nil {
				t.Fatalf("ParseIfNoneMatch error: %v", err)
			}
			if got := m.Matches(tt.etag, tt.weak); got != tt.want {
				t.Errorf("Matches(%q, %v) = %v, want %v", tt.etag, tt.weak, got, tt.want)
			}
		})
	}
}
