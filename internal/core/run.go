package core

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func RunUntilSignal(runtime *Runtime) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Start(ctx)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return <-errCh
}
