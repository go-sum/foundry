package headers

import "testing"

func TestParseVary(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStar   bool
		wantFields []string
	}{
		{
			name:     "wildcard",
			input:    "*",
			wantStar: true,
		},
		{
			name:       "single field",
			input:      "Accept",
			wantFields: []string{"Accept"},
		},
		{
			name:       "multiple fields",
			input:      "Accept, Accept-Encoding, Accept-Language",
			wantFields: []string{"Accept", "Accept-Encoding", "Accept-Language"},
		},
		{
			name:       "empty input",
			input:      "",
			wantFields: nil,
		},
		{
			name:       "case insensitive dedup",
			input:      "accept, Accept",
			wantFields: []string{"Accept"},
		},
		{
			name:       "lowercase fields get canonicalized",
			input:      "accept-encoding",
			wantFields: []string{"Accept-Encoding"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseVary(tt.input)
			if got.Star != tt.wantStar {
				t.Errorf("Star = %v, want %v", got.Star, tt.wantStar)
			}
			if len(got.Fields) != len(tt.wantFields) {
				t.Fatalf("Fields len = %d, want %d: %v", len(got.Fields), len(tt.wantFields), got.Fields)
			}
			for i, want := range tt.wantFields {
				if got.Fields[i] != want {
					t.Errorf("Fields[%d] = %q, want %q", i, got.Fields[i], want)
				}
			}
		})
	}
}

func TestVaryAdd(t *testing.T) {
	v := ParseVary("Accept")
	v2 := v.Add("Accept-Encoding")

	if !v2.Has("Accept") {
		t.Error("expected Accept to be present")
	}
	if !v2.Has("Accept-Encoding") {
		t.Error("expected Accept-Encoding to be present")
	}
	if len(v2.Fields) != 2 {
		t.Errorf("Fields len = %d, want 2", len(v2.Fields))
	}

	// Dedup: adding existing field returns same length.
	v3 := v2.Add("accept")
	if len(v3.Fields) != 2 {
		t.Errorf("dedup failed, Fields len = %d, want 2", len(v3.Fields))
	}

	// Star is not modified by Add.
	star := Vary{Star: true}
	star2 := star.Add("Accept")
	if !star2.Star {
		t.Error("Add to star Vary should return star")
	}
}

func TestVaryHas(t *testing.T) {
	v := ParseVary("Accept, Accept-Encoding")
	if !v.Has("accept") {
		t.Error("Has(accept) should be case-insensitive")
	}
	if !v.Has("ACCEPT-ENCODING") {
		t.Error("Has(ACCEPT-ENCODING) should be case-insensitive")
	}
	if v.Has("Content-Type") {
		t.Error("Has(Content-Type) should return false")
	}

	star := Vary{Star: true}
	if !star.Has("anything") {
		t.Error("star Vary.Has should return true for anything")
	}
}

func TestVaryString(t *testing.T) {
	star := Vary{Star: true}
	if star.String() != "*" {
		t.Errorf("star String() = %q, want *", star.String())
	}

	v := ParseVary("Accept, Accept-Encoding")
	got := v.String()
	want := "Accept, Accept-Encoding"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	empty := Vary{}
	if empty.String() != "" {
		t.Errorf("empty String() = %q, want empty", empty.String())
	}
}
