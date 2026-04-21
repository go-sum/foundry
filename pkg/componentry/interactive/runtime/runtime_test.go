package runtime_test

import (
	"strings"
	"testing"

	"github.com/go-sum/componentry/interactive/runtime"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestScript_nonEmpty(t *testing.T) {
	rendered := testutil.RenderNode(t, runtime.Script())
	if rendered == "" {
		t.Fatal("Script(): rendered output must not be empty")
	}
	if !strings.HasPrefix(rendered, "<script>") {
		t.Errorf("Script(): expected <script> tag, got: %s", rendered[:min(len(rendered), 40)])
	}
	if !strings.HasSuffix(rendered, "</script>") {
		t.Errorf("Script(): expected </script> closing tag")
	}
}

func TestScriptCSPHash_format(t *testing.T) {
	if runtime.ScriptCSPHash == "" {
		t.Fatal("ScriptCSPHash: must not be empty")
	}
	if !strings.HasPrefix(runtime.ScriptCSPHash, "'sha256-") {
		t.Errorf("ScriptCSPHash: expected format 'sha256-...', got: %s", runtime.ScriptCSPHash)
	}
	if !strings.HasSuffix(runtime.ScriptCSPHash, "'") {
		t.Errorf("ScriptCSPHash: expected closing quote, got: %s", runtime.ScriptCSPHash)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
