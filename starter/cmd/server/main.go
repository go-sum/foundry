package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-sum/foundry/internal/app"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.New(ctx)
	if err != nil {
		return fmt.Errorf("startup: %w", err)
	}
	defer func() {
		if closeErr := a.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
				return
			}
			err = errors.Join(err, fmt.Errorf("shutdown: %w", closeErr))
		}
	}()
	return a.Run(ctx)
}
