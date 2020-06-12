package main

import (
	"context"
	"os"
	"os/signal"
)

func CancelContextWithSignal(ctx context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	if len(signals) == 0 {
		return ctx, cancel
	}

	ch := make(chan os.Signal, len(signals))
	signal.Notify(ch, signals...)

	go func() {
		select {
		case <-ctx.Done():
		case <-ch:
		}

		cancel()
		signal.Stop(ch)
		close(ch)
	}()

	return ctx, cancel
}
