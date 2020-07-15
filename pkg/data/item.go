package data

import (
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

type Item struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Raw       []byte
}

func (item *Item) Clone() *Item {
	clone := new(Item)
	clone.CreatedAt = item.CreatedAt
	clone.UpdatedAt = item.UpdatedAt
	clone.Raw = make([]byte, len(item.Raw))
	copy(clone.Raw, item.Raw)
	return clone
}

func (item *Item) UnmarshalJSON(data []byte) error {
	if !gjson.ValidBytes(data) {
		return errors.New("invalid JSON")
	}

	item.Raw = make([]byte, len(data))
	copy(item.Raw, data)

	var err error
	if item.CreatedAt, err = getTime(data, "created_at"); err != nil {
		return err
	}
	if item.UpdatedAt, err = getTime(data, "updated_at"); err != nil {
		return err
	}
	return nil
}

func (item Item) MarshalJSON() ([]byte, error) {
	raw := item.Raw
	if len(raw) == 0 {
		raw = []byte("null")
	}
	return raw, nil
}

func getTime(json []byte, path string) (time.Time, error) {
	ts := gjson.GetBytes(json, path)
	if !ts.Exists() {
		return time.Time{}, errors.Errorf("JSON object does not have a value for path: %s", path)
	}

	return time.Parse(time.RFC3339, ts.String())
}
