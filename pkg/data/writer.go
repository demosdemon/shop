package data

import (
	"encoding/json"
	"io"
)

func NewWriter(w io.Writer) *Writer {
	return &Writer{encoder: json.NewEncoder(w)}
}

type Writer struct {
	encoder *json.Encoder
}

func (w *Writer) Write(item *Item) error {
	return w.encoder.Encode(item)
}
