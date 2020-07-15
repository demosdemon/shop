package shopify

import (
	"github.com/demosdemon/shop/pkg/data"
)

type PaginationResult struct {
	item *data.Item
	err  error
}

func (r PaginationResult) Item() *data.Item {
	return r.item
}

func (r PaginationResult) Err() error {
	return r.err
}
