package auth

import "testing"

func TestSanitizeReturnTo(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"/", "/"},
		{"/dashboard", "/dashboard"},
		{"/path?q=1", "/path?q=1"},
		{"//", "/"},
		{"//evil.com", "/"},
		{"https://evil.com", "/"},
		{"", "/"},
		{"/a\rb", "/"},
		{"/a\nb", "/"},
	}

	for _, tc := range cases {
		got := SanitizeReturnTo(tc.input)
		if got != tc.want {
			t.Errorf("SanitizeReturnTo(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
