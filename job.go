package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/demosdemon/multierrgroup"
	"github.com/hashicorp/go-multierror"
)

var elements = []string{
	"orders",
	"products",
	"customers",
}

type Job struct {
	StoreID  string
	Username string
	Password string
}

func (job *Job) Scrape(ctx context.Context, outputDir string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var g multierrgroup.Group
	for _, element := range elements {
		prefix := fmt.Sprintf("[%-19s][%-9s] ", job.StoreID, element)
		t := &task{
			ctx:       ctx,
			cancel:    cancel,
			element:   element,
			outputDir: outputDir,
			client: Client{
				StoreID:  job.StoreID,
				Username: job.Username,
				Password: job.Password,
				Logger:   NewLogger(LevelDebug, os.Stdout, prefix),
			},
		}
		g.Go(t.run)
	}
	return g.Wait()
}

type task struct {
	ctx       context.Context
	cancel    context.CancelFunc
	element   string
	outputDir string
	client    Client
}

func (t *task) run() error {
	output := filepath.Join(
		t.outputDir,
		t.client.StoreID,
		t.element+".jsonl",
	)

	log := t.client.Logger
	fp, first, last, err := t.getMinMaxUpdatedAt(output)
	if err != nil {
		log.Errorf("error scanning existing file: %v", err)
		return err
	}

	var ch <-chan PaginationResult

	if first.IsZero() && last.IsZero() {
		log.Infof("no existing data found, fetching all values")
		ch = t.client.Paginate(t.ctx, t.element, ListOptions{Limit: 250})
	} else {
		multiCh := make(chan PaginationResult)

		var wg sync.WaitGroup
		forward := func(options interface{}) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ch := t.client.Paginate(t.ctx, t.element, options)
				for v := range ch {
					multiCh <- v
				}
			}()
		}

		log.Infof("fetching all values before %s", first)
		forward(ListOptions{UpdatedAtMax: first, Limit: 250})

		log.Infof("fetching all values after %s", last)
		forward(ListOptions{UpdatedAtMin: last, Limit: 250})

		go func() {
			wg.Wait()
			close(multiCh)
		}()

		ch = multiCh
	}

	enc := json.NewEncoder(fp)
	for v := range ch {
		if v.err != nil {
			err = multierror.Append(err, v.err)
			continue
		}
		wErr := enc.Encode(v.msg)
		if wErr != nil {
			t.cancel()
			err = multierror.Append(err, wErr)
			log.Errorf("error writing record to file: %v", wErr)
		}
	}

	if cErr := fp.Close(); cErr != nil {
		err = multierror.Append(err, cErr)
	}

	if err == nil {
		log.Infof("finished fetching values with no errors")
	}

	return err
}

func (t *task) getMinMaxUpdatedAt(output string) (*os.File, time.Time, time.Time, error) {
	var first, last time.Time

	log := t.client.Logger
	log.Infof("scanning %q for oldest and latest updated_at timestamp", output)
	fp, err := os.OpenFile(output, os.O_RDWR, 0)
	if err != nil && os.IsNotExist(err) {
		t.client.Logger.Infof("%q does not exist, creating a new file", output)
		_ = os.MkdirAll(path.Dir(output), 0777)
		fp, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		return fp, first, last, err
	}
	if err != nil {
		return fp, first, last, err
	}

	dec := json.NewDecoder(fp)
	for dec.More() {
		var line map[string]json.RawMessage
		if err := dec.Decode(&line); err != nil {
			log.Errorf("invalid json in %q: %v", output, err)
			return fp, first, last, err
		}

		sUpdatedAtMsg := line["updated_at"]
		if sUpdatedAtMsg == nil {
			log.Warnf("missing `updated_at` key in record: %#v", line)
			continue
		}

		var t time.Time
		if err := json.Unmarshal(sUpdatedAtMsg, &t); err != nil {
			log.Warnf("invalid time format: %v", err)
			continue
		}

		if first.IsZero() || t.Before(first) {
			first = t
		}

		if last.IsZero() || t.After(last) {
			last = t
		}
	}

	return fp, first, last, nil
}
