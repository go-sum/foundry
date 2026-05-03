package worker

import (
	"context"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/pkg/web/logging"
)

func provideRuntime(_ context.Context) (Runtime, error) {
	cfg, err := config.LoadWorker()
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{
		Config: cfg,
		Logger: logging.New(logging.Config{Level: logging.ParseLogLevel(cfg.LogLevel)}),
	}, nil
}
