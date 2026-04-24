package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input     string
		want      Semver
		wantErr   bool
	}{
		{"v1.2.3", Semver{1, 2, 3, ""}, false},
		{"v0.0.0", Semver{0, 0, 0, ""}, false},
		{"v10.20.30", Semver{10, 20, 30, ""}, false},
		{"v1.2.3-rc.1", Semver{1, 2, 3, "-rc.1"}, false},
		{"v1.2.3.4", Semver{}, true},
		{"1.2.3", Semver{}, true},
		{"", Semver{}, true},
		{"vX.Y.Z", Semver{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBumpPatch(t *testing.T) {
	tests := []struct {
		input Semver
		want  Semver
	}{
		{Semver{1, 2, 3, ""}, Semver{1, 2, 4, ""}},
		{Semver{0, 0, 0, ""}, Semver{0, 0, 1, ""}},
		{Semver{1, 2, 3, "-rc.1"}, Semver{1, 2, 4, ""}},
	}

	for _, tt := range tests {
		got := tt.input.BumpPatch()
		if got != tt.want {
			t.Errorf("BumpPatch(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		input Semver
		want  string
	}{
		{Semver{1, 2, 3, ""}, "v1.2.3"},
		{Semver{0, 0, 0, ""}, "v0.0.0"},
		{Semver{1, 2, 3, "-rc.1"}, "v1.2.3-rc.1"},
	}

	for _, tt := range tests {
		got := tt.input.String()
		if got != tt.want {
			t.Errorf("String(%+v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGreaterThan(t *testing.T) {
	tests := []struct {
		a, b Semver
		want bool
	}{
		{Semver{1, 2, 4, ""}, Semver{1, 2, 3, ""}, true},
		{Semver{1, 3, 0, ""}, Semver{1, 2, 9, ""}, true},
		{Semver{2, 0, 0, ""}, Semver{1, 9, 9, ""}, true},
		{Semver{1, 2, 3, ""}, Semver{1, 2, 3, ""}, false},
		{Semver{1, 2, 2, ""}, Semver{1, 2, 3, ""}, false},
		{Semver{1, 1, 0, ""}, Semver{1, 2, 0, ""}, false},
	}

	for _, tt := range tests {
		got := tt.a.GreaterThan(tt.b)
		if got != tt.want {
			t.Errorf("GreaterThan(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
