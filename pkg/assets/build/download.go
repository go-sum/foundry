package build

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultClient is the HTTP client used for downloads.
var DefaultClient = &http.Client{Timeout: 30 * time.Second}

// ResolveVersion returns the version from the {NAME}_VERSION env var,
// falling back to defaultVersion.
func ResolveVersion(name, defaultVersion string) string {
	if v := strings.TrimSpace(os.Getenv(strings.ToUpper(name) + "_VERSION")); v != "" {
		return v
	}
	return defaultVersion
}

// FetchURL downloads url to target file, creating parent directories as needed.
// Skips if target already exists (skip-if-exists semantics).
// Returns (true, nil) if downloaded, (false, nil) if skipped.
func FetchURL(client *http.Client, url, target string, out io.Writer) (downloaded bool, err error) {
	if _, err := os.Stat(target); err == nil {
		fmt.Fprintf(out, "  ↷ %s: target exists, skipping\n", filepath.Base(target))
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, fmt.Errorf("mkdir %s: %w", filepath.Dir(target), err)
	}
	resp, err := client.Get(url)
	if err != nil {
		return false, fmt.Errorf("GET %s: %w", url, err)
	}
	defer closeOnReturn(&err, resp.Body, "response body for %s", url)
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	f, err := os.Create(target)
	if err != nil {
		return false, fmt.Errorf("create %s: %w", target, err)
	}
	defer closeOnReturn(&err, f, "file %s", target)
	if _, err := io.Copy(f, resp.Body); err != nil {
		return false, fmt.Errorf("write %s: %w", target, err)
	}
	return true, nil
}

func closeOnReturn(errp *error, closer io.Closer, subject string, args ...any) {
	if closeErr := closer.Close(); closeErr != nil && *errp == nil {
		*errp = fmt.Errorf("close %s: %w", fmt.Sprintf(subject, args...), closeErr)
	}
}
