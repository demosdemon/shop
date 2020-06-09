package main

import (
	"context"
	"log"

	"github.com/demosdemon/multierrgroup"
)

func main() {
	ch, err := LoadStores("./stores.jsonl")
	if err != nil {
		log.Fatal(err)
	}

	var g multierrgroup.Group
	for config := range ch {
		config := config
		fn := func() error {
			job, err := config.Job()
			if err != nil {
				return err
			}
			return job.Scrape(context.Background(), "./out")
		}
		g.Go(fn)
	}

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
