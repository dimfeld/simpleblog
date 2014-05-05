package cache

import (
	//"github.com/dimfeld/simpleblog/lru"
	"strings"
	"sync"
)

// Ideally this would have separate caches for small and large objects, so that loading a few large
// objects into the cache doesn't evict all the smaller ones.
type memoryCache struct {
	lock        sync.RWMutex
	memoryLimit int // Memory Limit in bytes. This isn't quite accurate.
	objectLimit int
	nextLevel   Cache
	object      map[string]Object
	memoryUsage int
}

func (m *memoryCache) Set(path string, item Object, writeThrough bool) error {
	m.lock.Lock()

	oldItem, ok := m.object[path]
	if ok {
		m.memoryUsage -= len(oldItem.Data)
	}

	if len(item.Data) > m.objectLimit {
		// If this item takes up more than 25% of our memory limit, don't store it in this cache,
		// but still do writethrough if enabled.
		delete(m.object, path)
	} else {
		if m.memoryUsage+len(item.Data) > m.memoryLimit {
			m.trim()
		}

		m.object[path] = item
		m.memoryUsage += len(item.Data)
	}
	m.lock.Unlock()

	if writeThrough && m.nextLevel != nil {
		m.nextLevel.Set(path, item, true)
	}

	return nil
}

func (m *memoryCache) Del(path string) {
	if strings.HasSuffix(path, "*") {
		// Delete all matching objects in the cache.
		prefix := path[0 : len(path)-1]
		m.lock.Lock()
		for key, item := range m.object {
			if strings.HasPrefix(key, prefix) {
				m.memoryUsage -= len(item.Data)
				delete(m.object, key)
			}
		}
		m.lock.Unlock()
	} else {
		m.lock.Lock()
		m.memoryUsage -= len(m.object[path].Data)
		delete(m.object, path)
		m.lock.Unlock()
	}

	if m.nextLevel != nil {
		m.nextLevel.Del(path)
	}
}

func (m *memoryCache) Get(path string, filler Filler) (item Object, err error) {
	m.lock.RLock()
	item = m.object[path]
	m.lock.RUnlock()

	if len(item.Data) == 0 {
		if m.nextLevel != nil {
			item, err = m.nextLevel.Get(path, filler)

			if err != nil && len(item.Data) < m.objectLimit {
				// We got a valid item. Add it to our cache.
				m.Set(path, item, false)
			}
		} else {
			item, err = filler.Fill(m, path)
		}
		if err != nil {
			return item, err
		}
	}

	return item, nil
}

// Trim memory usage of the array. Right now this just clears all the data, which is obviously
// non-optimal. This function assumes that we already have a write lock.
func (m *memoryCache) trim() {
	m.object = make(map[string]Object)
	m.memoryUsage = 0
}

// NewmemoryCache creates a new cache.
// 	memoryLimit is roughly the maximum amount of memory that will be used.
//  nextLevel is an optional interface allowing write-through behavior.
func NewMemoryCache(memoryLimit int, backing Filler, nextLevel Cache) Cache {
	return &memoryCache{memoryLimit: memoryLimit,
		objectLimit: memoryLimit / 4,
		object:      make(map[string]Object),
		nextLevel:   nextLevel}
}
