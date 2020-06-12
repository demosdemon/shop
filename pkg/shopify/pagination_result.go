package shopify

import (
	"encoding/json"
)

type PaginationResult struct {
	msg json.RawMessage
	err error
}

func (r PaginationResult) Message() json.RawMessage {
	return r.msg
}

func (r PaginationResult) Err() error {
	return r.err
}
