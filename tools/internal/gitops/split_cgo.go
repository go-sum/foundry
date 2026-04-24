//go:build cgo

package gitops

import (
	"fmt"
	"time"

	"github.com/splitsh/lite/splitter"
)

// SplitSubtree performs a subtree split for the given prefix using splitsh/lite
// and returns the resulting commit SHA.
func SplitSubtree(repoRoot, prefix string) (string, error) {
	cfg := &splitter.Config{
		Path:       repoRoot,
		Origin:     "HEAD",
		Prefixes:   []*splitter.Prefix{splitter.NewPrefix(prefix, "", nil)},
		GitVersion: "latest",
	}

	result := &splitter.Result{}

	if err := splitter.Split(cfg, result); err != nil {
		return "", fmt.Errorf("split %s: %w", prefix, err)
	}

	if result.Head() == nil {
		return "", fmt.Errorf("split %s: no commits produced", prefix)
	}

	_ = result.Duration(time.Millisecond) // consumed to avoid unused import

	return result.Head().String(), nil
}
