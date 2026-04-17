package webtest

import (
	"strings"
	"testing"
)

// AssertNoCRLF asserts that s contains no carriage return or line feed bytes.
func AssertNoCRLF(t *testing.T, label, s string) {
	t.Helper()
	if strings.ContainsAny(s, "\r\n") {
		t.Errorf("%s: contains CR or LF: %q", label, s)
	}
}

// AssertExactHTML asserts exact byte-for-byte HTML equality.
func AssertExactHTML(t *testing.T, want, got string) {
	t.Helper()
	if want != got {
		t.Errorf("HTML mismatch:\nwant: %q\n got: %q", want, got)
	}
}
