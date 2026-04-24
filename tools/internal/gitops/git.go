package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RepoRoot returns the root directory of the current git repository.
func RepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// EnsureCleanTree returns an error if the working tree has uncommitted changes.
func EnsureCleanTree(root string) error {
	cmd := exec.Command("git", "-c", "core.fileMode=false", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return fmt.Errorf("working tree is not clean:\n%s", string(out))
	}
	return nil
}

// EnsureOnBranch returns an error if the repository is not on the given branch.
func EnsureOnBranch(root, branch string) error {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git rev-parse: %w", err)
	}
	current := strings.TrimSpace(string(out))
	if current != branch {
		return fmt.Errorf("must be on branch %q, currently on %q", branch, current)
	}
	return nil
}

// Add stages the given files for commit.
func Add(root string, files ...string) error {
	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add %s: %w", strings.Join(files, " "), err)
	}
	return nil
}

// Commit creates a commit with the given message.
func Commit(root, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// Tag creates an annotated tag at HEAD.
func Tag(root, version string) error {
	cmd := exec.Command("git", "tag", version)
	cmd.Dir = root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git tag %s: %w", version, err)
	}
	return nil
}

// PushWithTags pushes the given branch and all tags to remote.
func PushWithTags(root, remote, branch string) error {
	cmd := exec.Command("git", "push", remote, branch, "--tags")
	cmd.Dir = root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push %s %s --tags: %w", remote, branch, err)
	}
	return nil
}

// CheckoutHead resets the given files to HEAD, discarding any working-tree changes.
func CheckoutHead(root string, files ...string) error {
	args := append([]string{"checkout", "HEAD", "--"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	return cmd.Run()
}
