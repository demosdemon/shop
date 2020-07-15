package job

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/hashicorp/go-multierror"

	"github.com/demosdemon/shop/internal/config"
	"github.com/demosdemon/shop/pkg/data"
	"github.com/demosdemon/shop/pkg/log"
	"github.com/demosdemon/shop/pkg/shopify"
)

type Job struct {
	log.Logger
	*config.Store
	*config.Runtime
	Client  *shopify.Client
	Element string
}

func (j *Job) Do(ctx context.Context) error {
	if j.Logger == nil {
		prefix := fmt.Sprintf("[%-21s][%-9s] ", j.StoreID, j.Element)
		j.Logger = log.NewLogger(log.LevelWarn, os.Stderr, prefix)
	}

	if j.Client == nil {
		j.Client = shopify.New(j.StoreID, j.Username, j.Password, shopify.WithLogger(j))
	}

	return j.do(ctx)
}

func (j *Job) do(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fp, first, last, err := j.getMinMaxUpdatedAt(ctx)
	if err != nil {
		j.Errorf("error scanning existing file: %v", err)
		return
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	results := make(chan *data.Item)
	defer close(results)

	wg.Add(1)
	go func() {
		defer wg.Done()
		w := data.NewWriter(fp)
		count := 0
		for v := range results {
			if wErr := w.Write(v); wErr != nil {
				j.Errorf("error writing record to file: %v", wErr)
				err = multierror.Append(err, wErr)
				cancel()
			}
			count++
		}

		if cErr := fp.Close(); cErr != nil {
			j.Errorf("error closing file: %v", cErr)
			err = multierror.Append(err, cErr)
		}

		j.Debugf("collected %d records", count)
	}()

	forward := func(options interface{}) error {
		if j.DryRun {
			v, _ := query.Values(options)
			s := v.Encode()
			j.Warnf("dry run enabled; would have paginated %s with options: %s", j.Element, s)
			return nil
		}

		ch, err := j.Client.Paginate(ctx, j.Element, options)
		if err != nil {
			j.Errorf("error starting pagination thread: %v", err)
			return err
		}
		for v := range ch {
			if err := v.Err(); err != nil {
				j.Errorf("error during pagination: %v", err)
				return err
			}
			results <- v.Item()
		}
		return nil
	}

	if first.IsZero() && last.IsZero() {
		j.Infof("no existing data found, fetching all %s", j.Element)
		if fErr := forward(shopify.ListOptions{Limit: 250}); fErr != nil {
			if err == nil {
				err = fErr
			}
			return err
		}
	} else {
		j.Infof("fetching all %s before %s", j.Element, fmtTime(first))
		if fErr := forward(shopify.ListOptions{UpdatedAtMax: first, Limit: 250}); fErr != nil {
			if err == nil {
				err = fErr
			}
			return err
		}

		j.Infof("fetching all %s after %s", j.Element, fmtTime(last))
		if fErr := forward(shopify.ListOptions{UpdatedAtMin: last, Limit: 250}); fErr != nil {
			if err == nil {
				err = fErr
			}
			return err
		}
	}

	return err
}

func (j *Job) getMinMaxUpdatedAt(ctx context.Context) (*os.File, time.Time, time.Time, error) {
	var first, last time.Time
	output := filepath.Join(
		j.OutputDirectory,
		j.StoreID,
		j.Element+".jsonl",
	)

	j.Infof("scanning %q for oldest and latest updated_at timestamp", output)
	fp, err := os.OpenFile(output, os.O_RDWR, 0)
	if err != nil && os.IsNotExist(err) {
		j.Infof("%q does not exist, creating a new file", output)
		_ = os.MkdirAll(path.Dir(output), 0777)
		fp, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		return fp, first, last, err
	}
	if err != nil {
		return fp, first, last, err
	}

	seen := 0
	r := data.NewReader(fp)
	for r.Scan() {
		select {
		case <-ctx.Done():
			_ = fp.Close()
			return nil, first, last, ctx.Err()
		default:
		}

		ts := r.Item().UpdatedAt

		if first.IsZero() || ts.Before(first) {
			first = ts
		}

		if last.IsZero() || ts.After(last) {
			last = ts
		}

		seen++
	}

	if err := r.Err(); err != nil {
		_ = fp.Close()
		return nil, first, last, err
	}

	j.Infof("scanned %d records, oldest %s, newest %s", seen, fmtTime(first), fmtTime(last))
	return fp, first, last, nil
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "(null)"
	}

	return t.Format(time.UnixDate)
}
