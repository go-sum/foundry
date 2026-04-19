package theme_test

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	testutil "github.com/go-sum/componentry/testutil"

	"github.com/go-sum/componentry/interactive/theme"
)

// cspHashOf computes the expected CSP hash for a raw JS string.
func cspHashOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// extractScriptContent strips the surrounding <script>…</script> tags.
func extractScriptContent(rendered string) string {
	inner := strings.TrimPrefix(rendered, "<script>")
	inner = strings.TrimSuffix(inner, "</script>")
	return inner
}

func TestThemeScript(t *testing.T) {
	tests := []struct {
		name string
		node g.Node
	}{
		{
			name: "theme script",
			node: theme.ThemeScript(),
		},
		{
			name: "selector script",
			node: theme.SelectorScript(),
		},
		{
			name: "theme selector default",
			node: theme.ThemeSelector(theme.ThemeSelectorProps{}),
		},
		{
			name: "theme selector custom icons",
			node: theme.ThemeSelector(theme.ThemeSelectorProps{
				LightIcon:  g.Text("sun"),
				DarkIcon:   g.Text("moon"),
				SystemIcon: g.Text("monitor"),
			}),
		},
		{
			name: "theme selector extra nodes",
			node: theme.ThemeSelector(theme.ThemeSelectorProps{
				Extra: []g.Node{g.Text("extra-content")},
			}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, tc.node)
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}

func TestScriptCSPHash(t *testing.T) {
	if theme.ScriptCSPHash == "" {
		t.Fatal("ScriptCSPHash: must not be empty")
	}
	if !strings.HasPrefix(theme.ScriptCSPHash, "'sha256-") {
		t.Errorf("ScriptCSPHash: expected format 'sha256-...', got: %s", theme.ScriptCSPHash)
	}
	if !strings.HasSuffix(theme.ScriptCSPHash, "'") {
		t.Errorf("ScriptCSPHash: expected trailing quote, got: %s", theme.ScriptCSPHash)
	}
	rendered := testutil.RenderNode(t, theme.ThemeScript())
	inner := extractScriptContent(rendered)
	want := cspHashOf(inner)
	if theme.ScriptCSPHash != want {
		t.Errorf("ScriptCSPHash mismatch:\n  got:  %s\n  want: %s", theme.ScriptCSPHash, want)
	}
}

func TestSelectorScriptCSPHash(t *testing.T) {
	if !strings.HasPrefix(theme.SelectorScriptCSPHash, "'sha256-") {
		t.Errorf("SelectorScriptCSPHash: expected format 'sha256-...', got: %s", theme.SelectorScriptCSPHash)
	}
	if !strings.HasSuffix(theme.SelectorScriptCSPHash, "'") {
		t.Errorf("SelectorScriptCSPHash: expected trailing quote, got: %s", theme.SelectorScriptCSPHash)
	}
	rendered := testutil.RenderNode(t, theme.SelectorScript())
	inner := extractScriptContent(rendered)
	want := cspHashOf(inner)
	if theme.SelectorScriptCSPHash != want {
		t.Errorf("SelectorScriptCSPHash mismatch:\n  got:  %s\n  want: %s", theme.SelectorScriptCSPHash, want)
	}
}
