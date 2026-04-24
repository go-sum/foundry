//go:build !cgo

package gitops

import "fmt"

func SplitSubtree(_, prefix string) (string, error) {
	return "", fmt.Errorf("subtree split requires libgit2 (CGO); run via: task ws:push / ws:release / ws:status / ws:deploy — these route through Docker automatically")
}
