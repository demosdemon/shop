package data

import (
	"time"

	"github.com/emirpasic/gods/sets/treeset"
)

func (set *Set) Iterator() Iterator {
	return Iterator{set.set.Iterator()}
}

type Iterator struct {
	treeset.Iterator
}

func (iterator *Iterator) Value() *Item {
	return iterator.Iterator.Value().(*Item)
}

func timeComparator(a, b time.Time) int {
	switch {
	case a.IsZero() && b.IsZero():
		return 0
	case a.IsZero() && !b.IsZero():
		return 1
	case !a.IsZero() && b.IsZero():
		return -1
	case a.Before(b):
		return -1
	case b.Before(a):
		return 1
	default:
		return 0
	}
}
