package main

import (
	"context"
	"flag"
	"fmt"
	_log "log"
	"os"
	"syscall"
	"time"

	"github.com/demosdemon/multierrgroup"

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
	parseFlags(&cfg)

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

func parseFlags(cfg *config.Runtime) {
	flag.StringVar(&cfg.StoresFile, "stores", "./stores.jsonl", "path to store configuration file")
	flag.StringVar(&cfg.OutputDirectory, "output", "./out", "output directory to store results")
	flag.BoolVar(&cfg.PeriodicStackDump, "stack", false, "periodically dump stack traces to `.trace` files in the output directory")
	flag.DurationVar(&cfg.StackDumpFrequency, "period", time.Minute, "duration between stack dumps")
	flag.BoolVar(&cfg.DryRun, "dryrun", false, "do not actually call shopify apis")
	flag.StringVar(&cfg.ShopifyAPIVersion, "shopify-version", shopify.DefaultAPIVersion, "shopify API version")
	flag.DurationVar(&cfg.HTTPTimeout, "timeout", shopify.DefaultHTTPTimeout, "http timeout per request (some requests may take a long time)")
	flag.IntVar(&cfg.HTTPRetryCount, "retries", shopify.DefaultRetryCount, "number of attempts to retry each HTTP request before failing")
	flag.DurationVar(&cfg.HTTPRetryDelay, "delay", shopify.DefaultRetryDelay, "minimum delay to wait before retrying after failures (rate limited errors are handled separately)")
	flag.DurationVar(&cfg.HTTPRetryJitter, "jitter", shopify.DefaultRetryJitter, "random jitter amount to add in to each wait period")
	flag.StringVar(&cfg.HTTPUserAgent, "user-agent", shopify.DefaultUserAgent, "user-agent to use in HTTP requests")
	flag.Parse()
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
