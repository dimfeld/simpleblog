package cache

import (
	"sort"
)

type SplitSizeChild struct {
	MaxSize int
	Cache   Cache
}

// A SplitSize contains multiple caches, and sends objects to different
// caches depending on the size of the item.
type SplitSize []SplitSizeChild

func NewSplitSize(children ...SplitSizeChild) SplitSize {
	s := SplitSize(children)
	sort.Sort(s)
	return s
}

func (c SplitSize) AddChildCache(maxSize int, cache Cache) SplitSize {
	c = append(c, SplitSizeChild{maxSize, cache})
	sort.Sort(c)
	return c
}

// Get an item from the cache, checking all caches.
func (c SplitSize) Get(path string, filler Filler) (o Object, err error) {
	for _, child := range c {
		o, err = child.Cache.Get(path, nil)
		if err == nil {
			return
		}
	}

	if err != nil {
		o, err = filler.Fill(c, path)
	}

	return
}

// Set adds an object to the appropriate cache for the size of the object.
// Note that it is valid for no cache to be large enough.
func (c SplitSize) Set(path string, object Object) error {
	objectSize := len(object.Data)
	for _, child := range c {
		if objectSize < child.MaxSize {
			return child.Cache.Set(path, object)
		}
	}

	// No cache is large enough to hold this object, but that's ok.
	return nil
}

func (c SplitSize) Del(path string) {
	for _, child := range c {
		child.Cache.Del(path)
	}
}

func (c SplitSize) Less(i, j int) bool {
	return c[i].MaxSize < c[j].MaxSize
}

func (c SplitSize) Len() int {
	return len(c)
}

func (c SplitSize) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
