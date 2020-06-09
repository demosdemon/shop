package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
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

	seen := make(map[string]bool)
	ch := make(chan *StoreConfig)
	go func() {
		defer close(ch)
		defer func() { _ = fp.Close() }()

		s := bufio.NewScanner(fp)
		for s.Scan() {
			r := bytes.NewReader(s.Bytes())
			d := json.NewDecoder(r)

			config := new(StoreConfig)
			if err := d.Decode(config); err != nil {
				log.Printf("Error decoding StoreConfig: %v", err)
				log.Println(s.Text())
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

		if err := s.Err(); err != nil && err != io.EOF {
			log.Printf("Error reading file (%s): %v", path, err)
		}
	}()

	return ch, nil
}
