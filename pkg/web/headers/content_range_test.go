package headers

import "testing"

func TestParseContentRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ContentRange
		wantErr bool
	}{
		{
			name:  "normal range",
			input: "bytes 0-499/1234",
			want:  ContentRange{Unit: "bytes", Start: 0, End: 499, Size: 1234},
		},
		{
			name:  "unsatisfied range",
			input: "bytes */1234",
			want:  ContentRange{Unit: "bytes", Start: -1, End: -1, Size: 1234},
		},
		{
			name:  "unknown size",
			input: "bytes 0-499/*",
			want:  ContentRange{Unit: "bytes", Start: 0, End: 499, Size: -1},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing unit",
			input:   "0-499/1234",
			wantErr: true,
		},
		{
			name:    "missing slash",
			input:   "bytes 0-499",
			wantErr: true,
		},
		{
			name:    "invalid size",
			input:   "bytes 0-499/abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseContentRange(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestContentRangeString(t *testing.T) {
	tests := []struct {
		input ContentRange
		want  string
	}{
		{ContentRange{Unit: "bytes", Start: 0, End: 499, Size: 1234}, "bytes 0-499/1234"},
		{ContentRange{Unit: "bytes", Start: -1, End: -1, Size: 1234}, "bytes */1234"},
		{ContentRange{Unit: "bytes", Start: 0, End: 499, Size: -1}, "bytes 0-499/*"},
	}

	for _, tt := range tests {
		got := tt.input.String()
		if got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}
