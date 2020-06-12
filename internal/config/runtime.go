package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_log "log"
	"os"
	"path"
	"runtime"
	"time"
)

type Runtime struct {
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
