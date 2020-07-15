package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/demosdemon/shop/internal/config"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		log.Fatal("usage: scrape <path to repo>")
	}

	repo := args[0]
	runtime := config.Runtime{RepositoryPath: repo}
	stores, err := runtime.ScanRepository()
	if err != nil {
		log.Fatal(err)
	}

	enc := json.NewEncoder(os.Stdout)
	for _, store := range stores {
		if err := enc.Encode(store); err != nil {
			log.Fatal(err)
		}
	}
}
