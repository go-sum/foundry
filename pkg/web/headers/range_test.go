package headers

import (
	"testing"
)

func int64Ptr(n int64) *int64 { return &n }

func TestParseRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Range
		wantErr bool
	}{
		{
			name:  "normal range",
			input: "bytes=0-499",
			want: Range{
				Unit: "bytes",
				Ranges: []ByteRange{
					{Start: int64Ptr(0), End: int64Ptr(499)},
				},
			},
		},
		{
			name:  "suffix range",
			input: "bytes=-500",
			want: Range{
				Unit: "bytes",
				Ranges: []ByteRange{
					{Start: nil, End: int64Ptr(500)},
				},
			},
		},
		{
			name:  "open end range",
			input: "bytes=500-",
			want: Range{
				Unit: "bytes",
				Ranges: []ByteRange{
					{Start: int64Ptr(500), End: nil},
				},
			},
		},
		{
			name:  "multi-range",
			input: "bytes=0-499, 600-999",
			want: Range{
				Unit: "bytes",
				Ranges: []ByteRange{
					{Start: int64Ptr(0), End: int64Ptr(499)},
					{Start: int64Ptr(600), End: int64Ptr(999)},
				},
			},
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "no equals sign",
			input:   "bytes 0-499",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRange(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Unit != tt.want.Unit {
				t.Errorf("Unit = %q, want %q", got.Unit, tt.want.Unit)
			}
			if len(got.Ranges) != len(tt.want.Ranges) {
				t.Fatalf("got %d ranges, want %d", len(got.Ranges), len(tt.want.Ranges))
			}
			for i, want := range tt.want.Ranges {
				got := got.Ranges[i]
				if (got.Start == nil) != (want.Start == nil) {
					t.Errorf("Ranges[%d].Start nil mismatch", i)
				} else if got.Start != nil && *got.Start != *want.Start {
					t.Errorf("Ranges[%d].Start = %d, want %d", i, *got.Start, *want.Start)
				}
				if (got.End == nil) != (want.End == nil) {
					t.Errorf("Ranges[%d].End nil mismatch", i)
				} else if got.End != nil && *got.End != *want.End {
					t.Errorf("Ranges[%d].End = %d, want %d", i, *got.End, *want.End)
				}
			}
		})
	}
}

func TestRangeCanSatisfy(t *testing.T) {
	tests := []struct {
		name  string
		input string
		size  int64
		want  bool
	}{
		{"normal range within bounds", "bytes=0-499", 1000, true},
		{"range beyond EOF", "bytes=2000-3000", 1000, false},
		{"suffix range last 500 of 1000", "bytes=-500", 1000, true},
		{"suffix range -0 not satisfiable", "bytes=-0", 1000, false},
		{"open end range", "bytes=500-", 1000, true},
		{"open end beyond size", "bytes=1000-", 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ParseRange(tt.input)
			if err != nil {
				t.Fatalf("ParseRange error: %v", err)
			}
			if got := r.CanSatisfy(tt.size); got != tt.want {
				t.Errorf("CanSatisfy(%d) = %v, want %v", tt.size, got, tt.want)
			}
		})
	}
}

func TestRangeNormalize(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		size      int64
		wantStart int64
		wantEnd   int64
		wantOK    bool
	}{
		{"normal range", "bytes=0-499", 1000, 0, 499, true},
		{"clamp end to size-1", "bytes=0-9999", 1000, 0, 999, true},
		{"suffix range", "bytes=-500", 1000, 500, 999, true},
		{"suffix exceeds size", "bytes=-5000", 1000, 0, 999, true},
		{"open end", "bytes=500-", 1000, 500, 999, true},
		{"out of bounds", "bytes=2000-3000", 1000, 0, 0, false},
		{"suffix -0 not satisfiable", "bytes=-0", 1000, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ParseRange(tt.input)
			if err != nil {
				t.Fatalf("ParseRange error: %v", err)
			}
			start, end, ok := r.Normalize(tt.size)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok {
				if start != tt.wantStart {
					t.Errorf("start = %d, want %d", start, tt.wantStart)
				}
				if end != tt.wantEnd {
					t.Errorf("end = %d, want %d", end, tt.wantEnd)
				}
			}
		})
	}
}

func TestRangeString(t *testing.T) {
	r := Range{Unit: "bytes", Ranges: []ByteRange{
		{Start: int64Ptr(0), End: int64Ptr(499)},
	}}
	want := "bytes=0-499"
	if got := r.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
