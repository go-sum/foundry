package headers

import (
	"testing"
)

func TestParseForwarded(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Forwarded
		wantErr bool
	}{
		{
			name:  "single element all fields",
			input: "for=192.0.2.60;proto=http;by=203.0.113.43;host=example.com",
			want: []Forwarded{
				{For: "192.0.2.60", Proto: "http", By: "203.0.113.43", Host: "example.com"},
			},
		},
		{
			name:  "multiple elements",
			input: "for=192.0.2.60, for=198.51.100.17;by=203.0.113.43",
			want: []Forwarded{
				{For: "192.0.2.60"},
				{For: "198.51.100.17", By: "203.0.113.43"},
			},
		},
		{
			name:  "quoted-string for IPv6",
			input: `for="[2001:db8::cafe]"`,
			want: []Forwarded{
				{For: "[2001:db8::cafe]"},
			},
		},
		{
			name:  "proto only",
			input: "proto=https",
			want: []Forwarded{
				{Proto: "https"},
			},
		},
		{
			name:  "IPv6 with port",
			input: `for="[::1]:4711"`,
			want: []Forwarded{
				{For: "[::1]:4711"},
			},
		},
		{
			name:  "unknown token as for value",
			input: "for=unknown;by=_hidden",
			want: []Forwarded{
				{For: "unknown", By: "_hidden"},
			},
		},
		{
			name:  "_hidden token as for",
			input: "for=_hidden",
			want: []Forwarded{
				{For: "_hidden"},
			},
		},
		{
			name:  "empty string returns empty slice",
			input: "",
			want:  []Forwarded{},
		},
		{
			name:    "malformed no equals sign",
			input:   "for",
			wantErr: true,
		},
		{
			name:  "whitespace around delimiters",
			input: "for = 192.0.2.1 ; proto = http",
			want: []Forwarded{
				{For: "192.0.2.1", Proto: "http"},
			},
		},
		{
			name:  "unknown key silently ignored",
			input: "for=1.2.3.4;unknown=value",
			want: []Forwarded{
				{For: "1.2.3.4"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseForwarded(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseForwarded(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseForwarded(%q) returned %d elements, want %d", tt.input, len(got), len(tt.want))
			}
			for i, w := range tt.want {
				g := got[i]
				if g.For != w.For {
					t.Errorf("element[%d].For = %q, want %q", i, g.For, w.For)
				}
				if g.By != w.By {
					t.Errorf("element[%d].By = %q, want %q", i, g.By, w.By)
				}
				if g.Host != w.Host {
					t.Errorf("element[%d].Host = %q, want %q", i, g.Host, w.Host)
				}
				if g.Proto != w.Proto {
					t.Errorf("element[%d].Proto = %q, want %q", i, g.Proto, w.Proto)
				}
			}
		})
	}
}
