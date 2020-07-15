package data

import (
	"encoding/json"
	"io"
)

func NewReader(r io.Reader) *Reader {
	return &Reader{decoder: json.NewDecoder(r)}
}

type Reader struct {
	decoder *json.Decoder
	error   error
	item    *Item
}

func (r *Reader) Scan() bool {
	if !r.decoder.More() {
		return false
	}
	if r.item == nil {
		r.item = new(Item)
	}
	r.error = r.decoder.Decode(r.item)
	return r.error == nil
}

func (r *Reader) Err() error {
	return r.error
}

func (r *Reader) Item() *Item {
	return r.item
}
