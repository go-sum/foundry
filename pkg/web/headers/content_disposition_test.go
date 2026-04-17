package headers

import "testing"

func TestParseContentDisposition(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantType    string
		wantName    string
		wantFile    string
		wantFileExt string
		wantErr     bool
	}{
		{
			name:     "attachment with filename",
			input:    `attachment; filename="foo.txt"`,
			wantType: "attachment",
			wantFile: "foo.txt",
		},
		{
			name:        "attachment with filename star",
			input:       `attachment; filename*=UTF-8''foo%20bar.txt`,
			wantType:    "attachment",
			wantFileExt: "foo bar.txt",
		},
		{
			name:        "both filename and filename* (ext preferred)",
			input:       `attachment; filename="foo.txt"; filename*=UTF-8''foo%20bar.txt`,
			wantType:    "attachment",
			wantFile:    "foo.txt",
			wantFileExt: "foo bar.txt",
		},
		{
			name:     "form-data with name and filename",
			input:    `form-data; name="file"; filename="upload.png"`,
			wantType: "form-data",
			wantName: "file",
			wantFile: "upload.png",
		},
		{
			name:     "inline",
			input:    "inline",
			wantType: "inline",
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd, err := ParseContentDisposition(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cd.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", cd.Type, tt.wantType)
			}
			if cd.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", cd.Name, tt.wantName)
			}
			if cd.Filename != tt.wantFile {
				t.Errorf("Filename = %q, want %q", cd.Filename, tt.wantFile)
			}
			if cd.FilenameExt != tt.wantFileExt {
				t.Errorf("FilenameExt = %q, want %q", cd.FilenameExt, tt.wantFileExt)
			}
		})
	}
}

func TestContentDispositionPreferredFilename(t *testing.T) {
	cd := ContentDisposition{Filename: "fallback.txt", FilenameExt: "preferred.txt"}
	if got := cd.PreferredFilename(); got != "preferred.txt" {
		t.Errorf("PreferredFilename() = %q, want %q", got, "preferred.txt")
	}

	cd2 := ContentDisposition{Filename: "only.txt"}
	if got := cd2.PreferredFilename(); got != "only.txt" {
		t.Errorf("PreferredFilename() = %q, want %q", got, "only.txt")
	}
}

func TestContentDispositionString(t *testing.T) {
	tests := []struct {
		name  string
		input ContentDisposition
		want  string
	}{
		{
			name:  "empty type",
			input: ContentDisposition{},
			want:  "",
		},
		{
			name:  "attachment with filename",
			input: ContentDisposition{Type: "attachment", Filename: "foo.txt"},
			want:  `attachment; filename=foo.txt`,
		},
		{
			name:  "form-data with name and filename",
			input: ContentDisposition{Type: "form-data", Name: "file", Filename: "upload.png"},
			want:  `form-data; name=file; filename=upload.png`,
		},
		{
			name:  "attachment with unicode filename",
			input: ContentDisposition{Type: "attachment", FilenameExt: "foo bar.txt"},
			want:  `attachment; filename*=UTF-8''foo%20bar.txt`,
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
