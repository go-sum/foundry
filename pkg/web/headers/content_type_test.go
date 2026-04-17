package headers

import "testing"

func TestParseContentType(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantMedia    string
		wantCharset  string
		wantBoundary string
		wantErr      bool
	}{
		{
			name:        "text/html with charset",
			input:       "text/html; charset=utf-8",
			wantMedia:   "text/html",
			wantCharset: "utf-8",
		},
		{
			name:         "multipart/form-data with boundary",
			input:        "multipart/form-data; boundary=--boundary",
			wantMedia:    "multipart/form-data",
			wantBoundary: "--boundary",
		},
		{
			name:      "bare application/json",
			input:     "application/json",
			wantMedia: "application/json",
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:        "quoted charset",
			input:       `text/html; charset="utf-8"`,
			wantMedia:   "text/html",
			wantCharset: "utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, err := ParseContentType(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ct.MediaType != tt.wantMedia {
				t.Errorf("MediaType = %q, want %q", ct.MediaType, tt.wantMedia)
			}
			if ct.Charset != tt.wantCharset {
				t.Errorf("Charset = %q, want %q", ct.Charset, tt.wantCharset)
			}
			if ct.Boundary != tt.wantBoundary {
				t.Errorf("Boundary = %q, want %q", ct.Boundary, tt.wantBoundary)
			}
		})
	}
}

func TestContentTypeString(t *testing.T) {
	tests := []struct {
		name  string
		input ContentType
		want  string
	}{
		{
			name:  "empty",
			input: ContentType{},
			want:  "",
		},
		{
			name:  "bare media type",
			input: ContentType{MediaType: "application/json"},
			want:  "application/json",
		},
		{
			name:  "with charset",
			input: ContentType{MediaType: "text/html", Charset: "utf-8"},
			want:  "text/html; charset=utf-8",
		},
		{
			name:  "with boundary",
			input: ContentType{MediaType: "multipart/form-data", Boundary: "--boundary"},
			want:  "multipart/form-data; boundary=--boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
