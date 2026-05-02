package static

import (
	"cmp"
	"path/filepath"
)

// AssetsConfig is the env-facing shape for static asset serving.
type AssetsConfig struct {
	PublicDir string `validate:"required"`
	URLPrefix string `validate:"required"`
}

// InitialAssetsConfig returns generic static asset defaults.
// publicDir overrides the default public directory; pass "" to use the default.
func InitialAssetsConfig(publicDir string) AssetsConfig {
	return AssetsConfig{
		PublicDir: cmp.Or(publicDir, filepath.Join("public", "static")),
		URLPrefix: "/static",
	}
}
