package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/assets/config"
)

func TestProcessSVG_basic(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M1 2h14"/></svg>`
	got, err := processSVG([]byte(svg), "arrow")
	if err != nil {
		t.Fatalf("processSVG: %v", err)
	}
	if !strings.Contains(got, `id="arrow"`) {
		t.Errorf("missing id=arrow in: %s", got)
	}
	if !strings.Contains(got, `viewBox="0 0 16 16"`) {
		t.Errorf("missing viewBox in: %s", got)
	}
	if !strings.Contains(got, `<path d="M1 2h14"/>`) {
		t.Errorf("missing inner path in: %s", got)
	}
}

func TestProcessSVG_stripsScript(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><script>alert('xss')</script><path d="M0 0"/></svg>`
	got, err := processSVG([]byte(svg), "icon")
	if err != nil {
		t.Fatalf("processSVG: %v", err)
	}
	if strings.Contains(got, "<script") {
		t.Errorf("script tag not stripped: %s", got)
	}
	if strings.Contains(got, "alert") {
		t.Errorf("script content not stripped: %s", got)
	}
}

func TestProcessSVG_stripsEventAttrs(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M0 0" onclick="evil()"/></svg>`
	got, err := processSVG([]byte(svg), "icon")
	if err != nil {
		t.Fatalf("processSVG: %v", err)
	}
	if strings.Contains(got, "onclick") {
		t.Errorf("onclick attr not stripped: %s", got)
	}
}

func TestProcessSVG_transfersPresentationAttrs(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M0 0"/></svg>`
	got, err := processSVG([]byte(svg), "icon")
	if err != nil {
		t.Fatalf("processSVG: %v", err)
	}
	if !strings.Contains(got, `fill="none"`) {
		t.Errorf("fill attr not transferred: %s", got)
	}
	if !strings.Contains(got, `stroke="currentColor"`) {
		t.Errorf("stroke attr not transferred: %s", got)
	}
}

func TestProcessSVG_empty(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"></svg>`
	got, err := processSVG([]byte(svg), "empty-icon")
	if err != nil {
		t.Fatalf("processSVG: %v", err)
	}
	if !strings.Contains(got, `<symbol id="empty-icon"`) {
		t.Errorf("missing symbol with id: %s", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), "/>") {
		t.Errorf("expected self-closing symbol for empty SVG: %s", got)
	}
}

func TestAllRemoteSources(t *testing.T) {
	tests := []struct {
		sources []config.SourcesConfig
		want    bool
	}{
		{
			sources: []config.SourcesConfig{
				{Path: "https://example.com/icons/"},
			},
			want: true,
		},
		{
			sources: []config.SourcesConfig{
				{Path: "https://example.com/icons/"},
				{Path: "http://cdn.example.com/"},
			},
			want: true,
		},
		{
			sources: []config.SourcesConfig{
				{Path: "https://example.com/icons/"},
				{Path: "static/icons/"},
			},
			want: false,
		},
		{
			sources: []config.SourcesConfig{
				{Path: "static/icons/"},
			},
			want: false,
		},
		{
			sources: []config.SourcesConfig{},
			want:    false,
		},
	}
	for _, tt := range tests {
		got := allRemoteSources(tt.sources)
		if got != tt.want {
			t.Errorf("allRemoteSources(%v) = %v, want %v", tt.sources, got, tt.want)
		}
	}
}

func TestBuildSprite_local(t *testing.T) {
	dir := t.TempDir()
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M5 12h14"/></svg>`
	if err := os.WriteFile(filepath.Join(dir, "arrow.svg"), []byte(svgContent), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	target := filepath.Join(outDir, "sprite.svg")

	cfg := config.SpriteConfig{
		Enabled: true,
		Target:  target,
		Sources: []config.SourcesConfig{
			{Path: dir + "/", Files: []string{"arrow.svg"}},
		},
	}

	var out strings.Builder
	if err := BuildSprite("test", cfg, DefaultClient, false, &out); err != nil {
		t.Fatalf("BuildSprite: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `id="arrow"`) {
		t.Errorf("missing arrow symbol in: %s", content)
	}
	if !strings.Contains(content, `viewBox="0 0 24 24"`) {
		t.Errorf("missing viewBox in: %s", content)
	}
}
