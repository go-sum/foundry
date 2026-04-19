package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sum/assets/config"
)

func TestRemoveStaleJS(t *testing.T) {
	dir := t.TempDir()

	managed := filepath.Join(dir, "app.js")
	stale := filepath.Join(dir, "old.js")

	if err := os.WriteFile(managed, []byte("// managed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("// stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		JS: config.JSConfig{
			Bundles: []config.JSBundle{
				{Entry: "src/app.js", Target: managed},
			},
		},
	}

	var out strings.Builder
	if err := RemoveStaleJS(cfg, &out); err != nil {
		t.Fatalf("RemoveStaleJS: %v", err)
	}

	if _, err := os.Stat(managed); err != nil {
		t.Errorf("managed file should still exist: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale file should have been removed")
	}
	if !strings.Contains(out.String(), "old.js") {
		t.Errorf("output should mention removed file, got: %s", out.String())
	}
}
