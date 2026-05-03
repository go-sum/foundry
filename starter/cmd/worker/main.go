package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-sum/foundry/internal/worker"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) (err error) {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w, err := worker.New(ctx)
	if err != nil {
		return fmt.Errorf("startup: %w", err)
	}
	defer func() {
		if closeErr := w.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
				return
			}
			err = errors.Join(err, fmt.Errorf("shutdown: %w", closeErr))
		}
	}()

	if len(args) > 0 && args[0] == "check" {
		return w.Check(ctx)
	}
	return w.Run(ctx)
}
