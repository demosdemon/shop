package config

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	_log "log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/demosdemon/shop/pkg/shopify"
)

type Runtime struct {
	RepositoryPath     string
	StoresFile         string
	OutputDirectory    string
	PeriodicStackDump  bool
	StackDumpFrequency time.Duration
	DryRun             bool
	ShopifyAPIVersion  string
	HTTPTimeout        time.Duration
	HTTPRetryCount     int
	HTTPRetryDelay     time.Duration
	HTTPRetryJitter    time.Duration
	HTTPUserAgent      string
}

func (r *Runtime) ParseArgs(args []string) error {
	name := args[0]
	name = filepath.Base(name)
	args = args[1:]

	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.StringVar(&r.StoresFile, "stores", "./stores.jsonl", "path to store configuration file")
	f.StringVar(&r.OutputDirectory, "output", "./out", "output directory to store results")
	f.BoolVar(&r.PeriodicStackDump, "stack", false, "periodically dump stack traces to `.trace` files in the output directory")
	f.DurationVar(&r.StackDumpFrequency, "period", time.Minute, "duration between stack dumps")
	f.BoolVar(&r.DryRun, "dryrun", false, "do not actually call shopify apis")
	f.StringVar(&r.ShopifyAPIVersion, "shopify-version", shopify.DefaultAPIVersion, "shopify API version")
	f.DurationVar(&r.HTTPTimeout, "timeout", shopify.DefaultHTTPTimeout, "http timeout per request (some requests may take a long time)")
	f.IntVar(&r.HTTPRetryCount, "retries", shopify.DefaultRetryCount, "number of attempts to retry each HTTP request before failing")
	f.DurationVar(&r.HTTPRetryDelay, "delay", shopify.DefaultRetryDelay, "minimum delay to wait before retrying after failures (rate limited errors are handled separately)")
	f.DurationVar(&r.HTTPRetryJitter, "jitter", shopify.DefaultRetryJitter, "random jitter amount to add in to each wait period")
	f.StringVar(&r.HTTPUserAgent, "user-agent", shopify.DefaultUserAgent, "user-agent to use in HTTP requests")
	return f.Parse(args)
}

func (r *Runtime) LoadStores() (<-chan *Store, error) {
	fp, err := os.Open(r.StoresFile)
	if err != nil {
		return nil, err
	}

	ch := make(chan *Store)
	go func() {
		dec := json.NewDecoder(fp)
		for dec.More() {
			config := new(Store)
			if err := dec.Decode(config); err != nil {
				_log.Printf("error decoding Store: %v", err)
				continue
			}
			ch <- config
		}

		close(ch)
		_ = fp.Close()
	}()

	return ch, nil
}

func (r *Runtime) PeriodicallyPrintStackDump(ctx context.Context) {
	if !r.PeriodicStackDump {
		return
	}

	_ = os.MkdirAll(r.OutputDirectory, 0777)
	t := time.NewTicker(r.StackDumpFrequency)
	defer t.Stop()

	errCount := 0
	count := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		if err := r.dumpStack(count); err != nil {
			_log.Printf("error writing stack dump: %v", err)
			if errCount > 5 {
				return
			}
			errCount++
		} else {
			errCount = 0
			count++
			if count >= 10_000 {
				count = 0
			}
		}
	}
}

func (r *Runtime) dumpStack(count int) error {
	stack := stack()
	fn := path.Join(r.OutputDirectory, fmt.Sprintf("shop-%04d.trace", count))
	return ioutil.WriteFile(fn, stack, 0666)
}

func stack() []byte {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}
