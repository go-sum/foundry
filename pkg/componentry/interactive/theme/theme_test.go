package theme_test

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"

	"github.com/go-sum/foundry/pkg/componentry/interactive/theme"
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
			name: "init script",
			node: theme.InitScript(),
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

func TestInitScriptCSPHash(t *testing.T) {
	if theme.InitScriptCSPHash == "" {
		t.Fatal("InitScriptCSPHash: must not be empty")
	}
	if !strings.HasPrefix(theme.InitScriptCSPHash, "'sha256-") {
		t.Errorf("InitScriptCSPHash: expected format 'sha256-...', got: %s", theme.InitScriptCSPHash)
	}
	rendered := testutil.RenderNode(t, theme.InitScript())
	inner := extractScriptContent(rendered)
	want := cspHashOf(inner)
	if theme.InitScriptCSPHash != want {
		t.Errorf("InitScriptCSPHash mismatch:\n  got:  %s\n  want: %s", theme.InitScriptCSPHash, want)
	}
}

