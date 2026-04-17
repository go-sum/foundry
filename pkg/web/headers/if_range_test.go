package headers

import (
	"net/http"
	"testing"
	"time"
)

func TestParseIfRange(t *testing.T) {
	refTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	refTimeStr := refTime.Format(http.TimeFormat)

	tests := []struct {
		name    string
		input   string
		wantTag string
		wantDate time.Time
		wantErr bool
	}{
		{
			name:    "strong ETag",
			input:   `"abc123"`,
			wantTag: "abc123",
		},
		{
			name:     "HTTP date",
			input:    refTimeStr,
			wantDate: refTime,
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
			name:    "invalid value",
			input:   "notadate-or-etag",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIfRange(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ETag != tt.wantTag {
				t.Errorf("ETag = %q, want %q", got.ETag, tt.wantTag)
			}
			if !tt.wantDate.IsZero() {
				if !got.Date.Equal(tt.wantDate) {
					t.Errorf("Date = %v, want %v", got.Date, tt.wantDate)
				}
			}
		})
	}
}

func TestIfRangeMatches(t *testing.T) {
	refTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		ifRange IfRange
		etag    string
		modTime time.Time
		want    bool
	}{
		{
			name:    "ETag match",
			ifRange: IfRange{ETag: "abc"},
			etag:    "abc",
			want:    true,
		},
		{
			name:    "ETag mismatch",
			ifRange: IfRange{ETag: "abc"},
			etag:    "xyz",
			want:    false,
		},
		{
			name:    "date match: resource not modified after",
			ifRange: IfRange{Date: refTime},
			modTime: refTime.Add(-time.Hour),
			want:    true,
		},
		{
			name:    "date mismatch: resource modified after",
			ifRange: IfRange{Date: refTime},
			modTime: refTime.Add(time.Hour),
			want:    false,
		},
		{
			name:    "empty IfRange returns false",
			ifRange: IfRange{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ifRange.Matches(tt.etag, tt.modTime)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIfRangeString(t *testing.T) {
	refTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	r := IfRange{ETag: "abc123"}
	if got := r.String(); got != `"abc123"` {
		t.Errorf("ETag String() = %q, want %q", got, `"abc123"`)
	}

	rd := IfRange{Date: refTime}
	want := refTime.Format(http.TimeFormat)
	if got := rd.String(); got != want {
		t.Errorf("Date String() = %q, want %q", got, want)
	}

	empty := IfRange{}
	if got := empty.String(); got != "" {
		t.Errorf("empty String() = %q, want empty", got)
	}
}
