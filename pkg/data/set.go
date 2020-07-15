package data

import (
	"github.com/emirpasic/gods/sets/treeset"
)

type Set struct {
	set *treeset.Set
}

func (set *Set) Init() *Set {
	set.set = treeset.NewWith(UpdatedAtComparator)
	return set
}

func (set *Set) Len() int {
	return set.set.Size()
}

func (set *Set) Add(items ...*Item) {
	data := make([]interface{}, len(items))
	for idx, item := range items {
		data[idx] = item
	}
	set.set.Add(data...)
}

func CreatedAtComparator(a, b interface{}) int {
	v1 := a.(*Item)
	v2 := b.(*Item)

	return timeComparator(v1.CreatedAt, v2.CreatedAt)
}

func UpdatedAtComparator(a, b interface{}) int {
	v1 := a.(*Item)
	v2 := b.(*Item)

	return timeComparator(v1.UpdatedAt, v2.UpdatedAt)
}
