package main

import (
	"fmt"
	"log"
	"os"

	"github.com/demosdemon/multierrgroup"
	"github.com/hashicorp/errwrap"

	"github.com/demosdemon/shop/pkg/data"
)

func main() {
	files := os.Args[1:]

	var g multierrgroup.Group
	for _, f := range files {
		f := f
		g.Go(reorder(f))
	}

	if err := g.Wait(); err != nil {
		if err, ok := err.(errwrap.Wrapper); ok {
			errs := err.WrappedErrors()
			log.Printf("%d errors occured:", len(errs))
			for _, err := range errs {
				log.Printf("* %v", err)
			}
			os.Exit(1)
		}
		log.Printf("fatal error: %v", err)
		os.Exit(2)
	}
}

type reorderError struct {
	path  string
	error error
}

func (e reorderError) Error() string {
	return fmt.Sprintf("error processing `%s`: %v", e.path, e.error)
}

func reorder(path string) func() error {
	return func() error {
		log.Printf("opening %s", path)
		fp, err := os.OpenFile(path, os.O_RDWR, 0666)
		if err != nil {
			return reorderError{path, err}
		}

		defer func() { _ = fp.Close() }()

		if err := data.Reorder(fp); err != nil {
			return reorderError{path, err}
		}

		log.Printf("finished %s", path)
		return nil
	}
}
