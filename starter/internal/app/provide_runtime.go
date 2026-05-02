package app

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-sum/foundry/pkg/assets/publish"
	"github.com/go-sum/foundry/pkg/componentry/assets/iconset"
	"github.com/go-sum/foundry/pkg/componentry/icons"
	"github.com/go-sum/foundry/pkg/web/logging"

	config "github.com/go-sum/foundry/config"
)

func provideRuntime(_ context.Context) (Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{
		Config: cfg,
		Logger: logging.New(logging.Config{Level: logging.ParseLogLevel(cfg.LogLevel)}),
		Tracer: noop.NewTracerProvider().Tracer("app"),
	}, nil
}

func provideAssets(cfg *config.Config) (*publish.Manifest, *icons.Registry, error) {
	manifest, err := publish.New(cfg.Assets.PublicDir, cfg.Assets.URLPrefix)
	if err != nil {
		return nil, nil, fmt.Errorf("assets: %w", err)
	}

	reg := publish.NewRegistry()
	reg.RegisterSprites(iconset.Default.Sprites)
	reg.SetPathFunc(manifest.Path)

	iconReg := icons.NewRegistry()
	resolved := make(map[icons.Key]icons.Ref, len(iconset.Default.Icons))
	for key, ref := range iconset.Default.Icons {
		resolved[key] = icons.Ref{
			Sprite: reg.SpritePath(ref.Sprite),
			ID:     ref.ID,
		}
	}
	iconReg.RegisterSet(resolved)
	return manifest, iconReg, nil
}

