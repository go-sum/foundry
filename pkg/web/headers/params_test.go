package headers

import (
	"reflect"
	"testing"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "single bare value",
			input: "charset=utf-8",
			want:  map[string]string{"charset": "utf-8"},
		},
		{
			name:  "quoted value",
			input: `charset="utf-8"`,
			want:  map[string]string{"charset": "utf-8"},
		},
		{
			name:  "multiple params",
			input: `charset="utf-8"; boundary=--abc`,
			want:  map[string]string{"charset": "utf-8", "boundary": "--abc"},
		},
		{
			name:  "backslash escape in quoted value",
			input: `filename="foo\"bar"`,
			want:  map[string]string{"filename": `foo"bar`},
		},
		{
			name:  "backslash backslash escape",
			input: `filename="foo\\bar"`,
			want:  map[string]string{"filename": `foo\bar`},
		},
		{
			name:  "whitespace around parts",
			input: `  key = value  `,
			want:  map[string]string{"key": "value"},
		},
		{
			name:  "key is lowercased",
			input: "Charset=UTF-8",
			want:  map[string]string{"charset": "UTF-8"},
		},
		{
			name:  "boolean token only (no equals) — skipped",
			input: "no-cache",
			want:  nil,
		},
		{
			name:  "mixed boolean and param",
			input: "no-cache; max-age=3600",
			want:  map[string]string{"max-age": "3600"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseParams(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseParams(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain value no quoting", "hello", "hello"},
		{"value with space", "hello world", `"hello world"`},
		{"value with semicolon", "a;b", `"a;b"`},
		{"value with comma", "a,b", `"a,b"`},
		{"value with double quote", `a"b`, `"a\"b"`},
		{"value with equals", "a=b", `"a=b"`},
		{"value with backslash", `a\b`, `"a\\b"`},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Quote(tt.input)
			if got != tt.want {
				t.Errorf("Quote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
