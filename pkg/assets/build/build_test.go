package build

import (
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if !opts.JS {
		t.Error("DefaultOptions.JS should be true")
	}
	if !opts.CSS {
		t.Error("DefaultOptions.CSS should be true")
	}
	if !opts.Fonts {
		t.Error("DefaultOptions.Fonts should be true")
	}
	if opts.Minify {
		t.Error("DefaultOptions.Minify should be false")
	}
}
