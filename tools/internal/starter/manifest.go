package starter

import (
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Manifest describes the clone rules loaded from manifest.yaml.
type Manifest struct {
	Exclude       []string      `yaml:"exclude"`
	Rename        []RenameRule  `yaml:"rename"`
	ModuleRewrite ModuleRewrite `yaml:"moduleRewrite"`
}

// RenameRule describes a single file rename operation applied after copying.
type RenameRule struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// ModuleRewrite holds the source module path to be replaced during clone.
type ModuleRewrite struct {
	From string `yaml:"from"`
}

// LoadManifest reads and parses the YAML manifest at the given path.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("LoadManifest: read %s: %w", path, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("LoadManifest: parse %s: %w", path, err)
	}
	return m, nil
}

// IsExcluded reports whether relPath should be excluded per the manifest exclude list.
func IsExcluded(manifest Manifest, path string) bool {
	if path == ".git" || strings.HasPrefix(path, ".git/") {
		return true
	}
	for _, ex := range manifest.Exclude {
		if ex == path {
			return true
		}
		// Directory pattern: ends with "/" — match everything inside.
		if strings.HasSuffix(ex, "/") && strings.HasPrefix(path, ex) {
			return true
		}
		// Directory without trailing slash: match the dir itself or anything inside.
		if !strings.Contains(ex, ".") && (path == ex || strings.HasPrefix(path, ex+"/")) {
			return true
		}
	}
	return false
}
