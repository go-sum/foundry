package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-sum/foundry/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.New(ctx)
	if err != nil {
		slog.Error("startup", "err", err)
		os.Exit(1)
	}

	startFailed := make(chan struct{})
	go func() {
		if err := a.Start(ctx); err != nil {
			slog.Error("serve", "err", err)
			close(startFailed)
			stop()
		}
	}()

	<-ctx.Done()

	sctx, cancel := context.WithTimeout(context.Background(), a.Runtime.Config.Server.ShutdownTimeout)
	defer cancel()

	if err := a.Shutdown(sctx); err != nil {
		slog.Error("shutdown", "err", err)
		os.Exit(1)
	}

	select {
	case <-startFailed:
		os.Exit(1)
	default:
	}
}
