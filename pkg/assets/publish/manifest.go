package publish

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Manifest maps asset names to content-hashed URLs.
type Manifest struct {
	manifest  map[string]string
	urlPrefix string
}

// Default is the package-level manifest used by Path.
var Default *Manifest

// New walks publicDir, hashes each file (SHA-256, first 8 hex chars),
// and builds a manifest keyed by relative path. A missing dir returns
// an empty manifest with no error.
func New(publicDir, prefix string) (*Manifest, error) {
	m := &Manifest{
		manifest:  make(map[string]string),
		urlPrefix: strings.TrimRight(prefix, "/"),
	}
	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		return m, nil
	}
	err := filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		hash, err := hashFile(path)
		if err != nil {
			return fmt.Errorf("hashing %s: %w", path, err)
		}
		rel, err := filepath.Rel(publicDir, path)
		if err != nil {
			return fmt.Errorf("rel path %s: %w", path, err)
		}
		rel = filepath.ToSlash(rel)
		m.manifest[rel] = hash
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", publicDir, err)
	}
	return m, nil
}

// Must panics if err is non-nil, otherwise returns m.
func Must(m *Manifest, err error) *Manifest {
	if err != nil {
		panic(err)
	}
	return m
}

// Path returns the URL for the named asset with a ?v= cache-busting suffix.
// If the name is not in the manifest, the bare URL is returned.
func (m *Manifest) Path(name string) string {
	name = strings.TrimPrefix(filepath.ToSlash(name), "/")
	url := m.urlPrefix + "/" + name
	if hash, ok := m.manifest[name]; ok {
		return url + "?v=" + hash
	}
	return url
}

// Init initializes the package-level Default manifest.
func Init(publicDir, prefix string) error {
	m, err := New(publicDir, prefix)
	if err != nil {
		return err
	}
	Default = m
	return nil
}

// MustInit initializes Default and panics on error.
func MustInit(publicDir, prefix string) {
	if err := Init(publicDir, prefix); err != nil {
		panic(err)
	}
}

// Path returns the URL for name from the Default manifest.
func Path(name string) string {
	if Default == nil {
		return "/" + strings.TrimPrefix(filepath.ToSlash(name), "/")
	}
	return Default.Path(name)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:8], nil
}
