package demo_test

import (
	"io"
	"strings"
	"testing"

	"github.com/go-sum/showcase/componentry/demo"
	g "maragu.dev/gomponents"
)

func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(io.Writer(&b)); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

func TestSearchResults_Empty(t *testing.T) {
	out := render(t, demo.SearchResults(""))
	if !strings.Contains(out, "Alice Johnson") {
		t.Errorf("expected all rows when query is empty; Alice Johnson not found")
	}
	if !strings.Contains(out, "Bob Smith") {
		t.Errorf("expected all rows when query is empty; Bob Smith not found")
	}
}

func TestSearchResults_Filtered(t *testing.T) {
	out := render(t, demo.SearchResults("alice"))
	if !strings.Contains(out, "Alice Johnson") {
		t.Errorf("expected Alice Johnson in filtered results")
	}
	if strings.Contains(out, "Bob Smith") {
		t.Errorf("Bob Smith should not appear in 'alice' search")
	}
}

func TestSearchResults_NoMatch(t *testing.T) {
	out := render(t, demo.SearchResults("zzznomatch"))
	if !strings.Contains(out, "No results found") {
		t.Errorf("expected 'No results found' message")
	}
}

func TestValidationResult_EmailValid(t *testing.T) {
	out := render(t, demo.ValidationResult("email", "user@example.com"))
	if !strings.Contains(out, "Looks good") {
		t.Errorf("expected 'Looks good' for valid email")
	}
}

func TestValidationResult_EmailInvalid(t *testing.T) {
	out := render(t, demo.ValidationResult("email", "notanemail"))
	if !strings.Contains(out, "valid email") {
		t.Errorf("expected error message for invalid email")
	}
}

func TestValidationResult_UsernameShort(t *testing.T) {
	out := render(t, demo.ValidationResult("username", "ab"))
	if !strings.Contains(out, "3 characters") {
		t.Errorf("expected length error for short username")
	}
}

func TestPaginatedTable_FirstPage(t *testing.T) {
	out := render(t, demo.PaginatedTable(1, 10))
	if !strings.Contains(out, "paginate-region") {
		t.Errorf("expected paginate-region id in output")
	}
	if !strings.Contains(out, "Page 1 of") {
		t.Errorf("expected page indicator in output")
	}
}

func TestPaginatedTable_LastPage(t *testing.T) {
	out := render(t, demo.PaginatedTable(3, 10))
	if !strings.Contains(out, "Page 3 of 3") {
		t.Errorf("expected Page 3 of 3")
	}
}

func TestRegionOptions_Known(t *testing.T) {
	tests := []struct {
		id      string
		contain string
	}{
		{"se", "Stockholm"},
		{"us", "California"},
		{"de", "Berlin"},
	}
	for _, tc := range tests {
		out := render(t, demo.RegionOptions(tc.id))
		if !strings.Contains(out, tc.contain) {
			t.Errorf("id=%q: expected %q in output\ngot: %s", tc.id, tc.contain, out)
		}
	}
}

func TestRegionOptions_Unknown(t *testing.T) {
	out := render(t, demo.RegionOptions("xx"))
	if !strings.Contains(out, "No regions available") {
		t.Errorf("expected fallback message for unknown country")
	}
}

func TestPathConstants(t *testing.T) {
	paths := []string{demo.PathSearch, demo.PathValidate, demo.PathPaginate, demo.PathRegion}
	for _, p := range paths {
		if !strings.HasPrefix(p, "/showcase/componentry/") {
			t.Errorf("path constant %q missing /showcase/componentry/ prefix", p)
		}
	}
}
