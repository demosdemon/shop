package main

import (
	"context"
	"flag"
	"fmt"
	_log "log"
	"os"
	"syscall"

	"github.com/demosdemon/multierrgroup"
	"github.com/pkg/errors"

	"github.com/demosdemon/shop/internal/config"
	"github.com/demosdemon/shop/internal/job"
	"github.com/demosdemon/shop/pkg/log"
	"github.com/demosdemon/shop/pkg/shopify"
)

var elements = []string{
	"orders",
	"products",
	"customers",
}

func main() {
	var cfg config.Runtime
	if err := cfg.ParseArgs(os.Args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// help has already been displayed
			os.Exit(1)
		}

		_log.Fatal(err)
	}

	ch, err := cfg.LoadStores()
	if err != nil {
		_log.Fatal(err)
	}

	ctx, cancel := CancelContextWithSignal(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go cfg.PeriodicallyPrintStackDump(ctx)

	var g multierrgroup.Group
	for store := range ch {
		g.Go(do(ctx, store, &cfg))
	}

	if err := g.Wait(); err != nil {
		_log.Fatal(err)
	}
}

func do(ctx context.Context, store *config.Store, runtime *config.Runtime) func() error {
	client := shopify.New(
		store.StoreID,
		store.Username,
		store.Password,
		shopify.WithAPIVersion(runtime.ShopifyAPIVersion),
		shopify.WithHTTPTimeout(runtime.HTTPTimeout),
		shopify.WithRetryCount(runtime.HTTPRetryCount),
		shopify.WithRetryDelay(runtime.HTTPRetryDelay),
		shopify.WithRetryJitter(runtime.HTTPRetryJitter),
		shopify.WithUserAgent(runtime.HTTPUserAgent),
	)

	return func() error {
		for _, element := range elements {
			prefix := fmt.Sprintf("[%-21s][%-9s] ", store.StoreID, element)
			logger := log.NewLogger(log.LevelDebug, os.Stderr, prefix)
			shopify.WithLogger(logger)(client)
			j := job.Job{
				Logger:  logger,
				Store:   store,
				Runtime: runtime,
				Client:  client,
				Element: element,
			}
			if err := j.Do(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}
