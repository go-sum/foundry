package starter

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// RunVerify clones the foundry starter into a temp directory, then runs
// go build ./starter/cmd/server and go vet ./... to verify the template is healthy.
func RunVerify(source string, w io.Writer) error {
	if source == "" {
		var err error
		source, err = findSourceRoot()
		if err != nil {
			return err
		}
	}

	tmpDir, err := os.MkdirTemp("", "foundry-verify-*")
	if err != nil {
		return fmt.Errorf("verify: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	fmt.Fprintf(w, "Cloning into %s...\n", tmpDir)

	// MkdirTemp creates the dir; clone rejects non-empty or existing dirs.
	// Remove and recreate so clone can write into a clean directory.
	if err := os.Remove(tmpDir); err != nil {
		return fmt.Errorf("verify: remove temp dir: %w", err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("verify: recreate temp dir: %w", err)
	}

	opts := CloneOptions{
		Source: source,
		Target: tmpDir,
		Module: "example.com/verify",
	}
	if err := RunClone(opts, w); err != nil {
		return fmt.Errorf("verify: clone failed: %w", err)
	}

	allPassed := true

	fmt.Fprintf(w, "\n[1/2] go build ./starter/cmd/server ... ")
	buildCmd := exec.Command("go", "build", "./starter/cmd/server")
	buildCmd.Dir = tmpDir
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(w, "FAIL\n")
		if len(buildOut) > 0 {
			fmt.Fprintf(w, "%s\n", buildOut)
		}
		allPassed = false
	} else {
		fmt.Fprintf(w, "PASS\n")
	}

	fmt.Fprintf(w, "[2/2] go vet ./...               ... ")
	vetCmd := exec.Command("go", "vet", "./...")
	vetCmd.Dir = tmpDir
	vetOut, err := vetCmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(w, "FAIL\n")
		if len(vetOut) > 0 {
			fmt.Fprintf(w, "%s\n", vetOut)
		}
		allPassed = false
	} else {
		fmt.Fprintf(w, "PASS\n")
	}

	fmt.Fprintln(w)
	if !allPassed {
		fmt.Fprintf(w, "Result: FAIL\n")
		return fmt.Errorf("verify: one or more checks failed")
	}
	fmt.Fprintf(w, "Result: PASS\n")
	return nil
}
