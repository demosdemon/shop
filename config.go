package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/pkg/errors"
)

type StoreConfig struct {
	File    string `json:"file"`
	ID      string `json:"id"`
	Options struct {
		Name     string `json:"name"`
		ID       string `json:"id"`
		Username string `json:"key"`
		Password string `json:"password"`
	} `json:"options"`
}

func (config *StoreConfig) StoreID() string {
	id := config.Options.Name
	if id == "" {
		id = config.Options.ID
	}

	return id
}

func (config *StoreConfig) Validate() error {
	if config == nil {
		return errors.New("config == nil")
	}

	if config.Options.Username == "" {
		return errors.New(`username == ""`)
	}

	if config.Options.Password == "" {
		return errors.New(`password == ""`)
	}

	if config.StoreID() == "" {
		return errors.New(`name and id == ""`)
	}

	return nil
}

func (config *StoreConfig) Job() (*Job, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Job{
		StoreID:  config.StoreID(),
		Username: config.Options.Username,
		Password: config.Options.Password,
	}, nil
}

func LoadStores(path string) (<-chan *StoreConfig, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	ch := make(chan *StoreConfig)
	go func() {
		seen := make(map[string]bool)
		dec := json.NewDecoder(fp)
		for dec.More() {
			config := new(StoreConfig)
			if err := dec.Decode(config); err != nil {
				log.Printf("error decoding StoreConfig: %v", err)
				continue
			}

			if err := config.Validate(); err != nil {
				continue
			}

			storeID := config.StoreID()
			if seen[storeID] {
				continue
			}

			seen[storeID] = true
			ch <- config
		}

		close(ch)
		_ = fp.Close()
	}()

	return ch, nil
}
