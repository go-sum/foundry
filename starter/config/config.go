package config

import (
	"fmt"

	cfgpkg "github.com/go-sum/config"
)

type Config struct {
	Env      cfgpkg.Env
	Security SecureConfig
}

var envOverlays = map[cfgpkg.Env]func(*Config){
	cfgpkg.Development: devOverlay,
	cfgpkg.Testing:     testOverlay,
}

func Load() (*Config, error) {
	cfg, err := productionDefault()
	if err != nil {
		return nil, err
	}
	cfg.Env = cfgpkg.CurrentEnv()

	if apply, ok := envOverlays[cfg.Env]; ok {
		apply(&cfg)
	}

	if err := cfgpkg.Validate(cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func productionDefault() (Config, error) {
	sec, err := defaultSecure()
	if err != nil {
		return Config{}, fmt.Errorf("config: security: %w", err)
	}
	return Config{
		Env:      cfgpkg.Production,
		Security: sec,
	}, nil
}
