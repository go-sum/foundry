package font

import (
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/testutil"

	g "maragu.dev/gomponents"
)

// renderNodes renders a []g.Node slice into a single HTML string.
func renderNodes(t *testing.T, nodes []g.Node) string {
	t.Helper()
	if len(nodes) == 0 {
		return ""
	}
	return testutil.RenderNode(t, g.Group(nodes))
}

// assertContains fails the test if any want string is absent from got.
func assertContains(t *testing.T, got string, want []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in rendered output:\n%s", w, got)
		}
	}
}

func TestGoogle(t *testing.T) {
	tests := []struct {
		name         string
		families     []string
		wantContains []string
		wantAbsent   []string
		wantStyle    []string
		wantFont     []string
	}{
		{
			name:     "two families produce correct URL with space encoding",
			families: []string{"Inter", "Roboto Mono"},
			wantContains: []string{
				`rel="preconnect" href="https://fonts.googleapis.com"`,
				`rel="preconnect" href="https://fonts.gstatic.com"`,
				`crossorigin`,
				`rel="stylesheet"`,
				`fonts.googleapis.com/css2?family=Inter&amp;family=Roboto+Mono&amp;display=swap`,
			},
			wantStyle: []string{"https://fonts.googleapis.com"},
			wantFont:  []string{"https://fonts.gstatic.com"},
		},
		{
			name:     "no families produces no nodes",
			families: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Google(tc.families...)
			nodes := p.Nodes()

			if len(tc.families) == 0 {
				if len(nodes) != 0 {
					t.Fatalf("want 0 nodes, got %d", len(nodes))
				}
				srcs := p.CSPSources()
				if len(srcs.StyleSrc) != 0 || len(srcs.FontSrc) != 0 {
					t.Fatalf("want empty CSPSources, got %+v", srcs)
				}
				return
			}

			got := renderNodes(t, nodes)
			assertContains(t, got, tc.wantContains)

			srcs := p.CSPSources()
			for _, want := range tc.wantStyle {
				if !containsStr(srcs.StyleSrc, want) {
					t.Fatalf("CSPSources.StyleSrc missing %q, got %v", want, srcs.StyleSrc)
				}
			}
			for _, want := range tc.wantFont {
				if !containsStr(srcs.FontSrc, want) {
					t.Fatalf("CSPSources.FontSrc missing %q, got %v", want, srcs.FontSrc)
				}
			}
		})
	}
}

func TestBunny(t *testing.T) {
	tests := []struct {
		name         string
		families     []string
		wantContains []string
		wantStyle    []string
		wantFont     []string
	}{
		{
			name:     "single family produces correct URL and preconnect",
			families: []string{"Inter"},
			wantContains: []string{
				`rel="preconnect" href="https://fonts.bunny.net"`,
				`rel="stylesheet"`,
				`fonts.bunny.net/css?family=Inter&amp;display=swap`,
			},
			wantStyle: []string{"https://fonts.bunny.net"},
			wantFont:  []string{"https://fonts.bunny.net"},
		},
		{
			name:     "no families produces no nodes",
			families: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Bunny(tc.families...)
			nodes := p.Nodes()

			if len(tc.families) == 0 {
				if len(nodes) != 0 {
					t.Fatalf("want 0 nodes, got %d", len(nodes))
				}
				srcs := p.CSPSources()
				if len(srcs.StyleSrc) != 0 || len(srcs.FontSrc) != 0 {
					t.Fatalf("want empty CSPSources, got %+v", srcs)
				}
				return
			}

			got := renderNodes(t, nodes)
			assertContains(t, got, tc.wantContains)

			srcs := p.CSPSources()
			for _, want := range tc.wantStyle {
				if !containsStr(srcs.StyleSrc, want) {
					t.Fatalf("CSPSources.StyleSrc missing %q, got %v", want, srcs.StyleSrc)
				}
			}
			for _, want := range tc.wantFont {
				if !containsStr(srcs.FontSrc, want) {
					t.Fatalf("CSPSources.FontSrc missing %q, got %v", want, srcs.FontSrc)
				}
			}
		})
	}
}

func TestAdobe(t *testing.T) {
	tests := []struct {
		name         string
		projectID    string
		wantContains []string
		wantStyle    []string
		wantFont     []string
	}{
		{
			name:      "project ID produces correct stylesheet URL",
			projectID: "abc123",
			wantContains: []string{
				`rel="stylesheet" href="https://use.typekit.net/abc123.css"`,
			},
			wantStyle: []string{"https://use.typekit.net"},
			wantFont:  []string{"https://use.typekit.net", "https://p.typekit.net"},
		},
		{
			name:      "empty project ID produces no nodes",
			projectID: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Adobe(tc.projectID)
			nodes := p.Nodes()

			if tc.projectID == "" {
				if len(nodes) != 0 {
					t.Fatalf("want 0 nodes, got %d", len(nodes))
				}
				srcs := p.CSPSources()
				if len(srcs.StyleSrc) != 0 || len(srcs.FontSrc) != 0 {
					t.Fatalf("want empty CSPSources, got %+v", srcs)
				}
				return
			}

			got := renderNodes(t, nodes)
			assertContains(t, got, tc.wantContains)

			srcs := p.CSPSources()
			for _, want := range tc.wantStyle {
				if !containsStr(srcs.StyleSrc, want) {
					t.Fatalf("CSPSources.StyleSrc missing %q, got %v", want, srcs.StyleSrc)
				}
			}
			for _, want := range tc.wantFont {
				if !containsStr(srcs.FontSrc, want) {
					t.Fatalf("CSPSources.FontSrc missing %q, got %v", want, srcs.FontSrc)
				}
			}
		})
	}
}

func TestSelf(t *testing.T) {
	tests := []struct {
		name         string
		faces        []Face
		wantContains []string
		wantCSPEmpty bool
	}{
		{
			name: "one face emits preload and font-face",
			faces: []Face{
				{Family: "Inter", Src: "/fonts/inter.woff2", Style: "normal", Weight: "400", Display: "swap"},
			},
			wantContains: []string{
				`rel="preload"`,
				`as="font"`,
				`href="/fonts/inter.woff2"`,
				`crossorigin`,
				`@font-face`,
				`font-family: 'Inter'`,
				`src: url('/fonts/inter.woff2')`,
				`font-style: normal`,
				`font-weight: 400`,
				`font-display: swap`,
			},
			wantCSPEmpty: true,
		},
		{
			name: "Display defaults to swap, Style to normal, Weight to 400",
			faces: []Face{
				{Family: "MyFont", Src: "/fonts/myfont.woff2"},
			},
			wantContains: []string{
				`font-display: swap`,
				`font-style: normal`,
				`font-weight: 400`,
			},
			wantCSPEmpty: true,
		},
		{
			name:         "no faces produces no nodes",
			faces:        nil,
			wantCSPEmpty: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Self(tc.faces...)
			nodes := p.Nodes()

			if len(tc.faces) == 0 {
				if len(nodes) != 0 {
					t.Fatalf("want 0 nodes, got %d", len(nodes))
				}
			} else {
				got := renderNodes(t, nodes)
				assertContains(t, got, tc.wantContains)
			}

			srcs := p.CSPSources()
			if tc.wantCSPEmpty {
				if len(srcs.StyleSrc) != 0 || len(srcs.FontSrc) != 0 {
					t.Fatalf("want empty CSPSources, got %+v", srcs)
				}
			}
		})
	}
}

func TestCollectCSPSources(t *testing.T) {
	t.Run("merges and deduplicates across providers", func(t *testing.T) {
		srcs := CollectCSPSources(
			Google("Inter"),
			Bunny("Inter"),
			Adobe("abc123"),
		)

		wantStyle := []string{"https://fonts.googleapis.com", "https://fonts.bunny.net", "https://use.typekit.net"}
		for _, want := range wantStyle {
			if !containsStr(srcs.StyleSrc, want) {
				t.Fatalf("StyleSrc missing %q, got %v", want, srcs.StyleSrc)
			}
		}

		wantFont := []string{"https://fonts.gstatic.com", "https://fonts.bunny.net", "https://use.typekit.net", "https://p.typekit.net"}
		for _, want := range wantFont {
			if !containsStr(srcs.FontSrc, want) {
				t.Fatalf("FontSrc missing %q, got %v", want, srcs.FontSrc)
			}
		}
	})

	t.Run("deduplicates repeated sources", func(t *testing.T) {
		srcs := CollectCSPSources(
			Google("Inter"),
			Google("Roboto"),
		)
		styleCount := 0
		for _, s := range srcs.StyleSrc {
			if s == "https://fonts.googleapis.com" {
				styleCount++
			}
		}
		if styleCount != 1 {
			t.Fatalf("expected deduplicated StyleSrc, found %d occurrences of googleapis.com in %v", styleCount, srcs.StyleSrc)
		}
	})

	t.Run("no providers returns empty CSPSources", func(t *testing.T) {
		srcs := CollectCSPSources()
		if len(srcs.StyleSrc) != 0 || len(srcs.FontSrc) != 0 {
			t.Fatalf("want empty CSPSources, got %+v", srcs)
		}
	})
}

func TestNodes(t *testing.T) {
	t.Run("concatenates all provider nodes", func(t *testing.T) {
		nodes := Nodes(
			Google("Inter"),
			Bunny("Inter"),
		)
		got := renderNodes(t, nodes)
		assertContains(t, got, []string{
			"fonts.googleapis.com",
			"fonts.bunny.net",
		})
	})

	t.Run("no providers returns nil", func(t *testing.T) {
		nodes := Nodes()
		if nodes != nil {
			t.Fatalf("want nil, got %v", nodes)
		}
	})
}

// containsStr reports whether slice contains s.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
