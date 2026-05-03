package app

import (
	"testing"

	configpkg "github.com/go-sum/foundry/config"
)

func TestNeedsKV_AlwaysTrueForWebRuntime(t *testing.T) {
	cfg := &configpkg.Config{}
	if !needsKV(cfg) {
		t.Fatal("needsKV() = false, want true")
	}
}
