package cache

import (
	//"github.com/dimfeld/simpleblog/lru"
	"strings"
	"sync"
)

type memoryCache struct {
	lock        sync.RWMutex
	memoryLimit int // Memory Limit in bytes. This isn't quite accurate.
	backing     Getter
	nextLevel   Cache
	item        map[string][]byte
	memoryUsage int
}

func (m *memoryCache) Set(path string, object []byte) {
	m.set(path, object, true)
}

func (m *memoryCache) set(path string, object []byte, writeThrough bool) {
	m.lock.Lock()

	oldItem := m.item[path]
	if len(oldItem) != 0 {
		m.memoryUsage -= len(oldItem)
	}

	if m.memoryUsage+len(object) > m.memoryLimit {
		m.trim()
	}

	m.item[path] = object
	m.memoryUsage += len(object)
	m.lock.Unlock()

	if writeThrough && m.nextLevel != nil {
		m.nextLevel.Set(path, object)
	}
}

func (m *memoryCache) Del(path string) {
	if strings.HasSuffix(path, "*") {
		// Delete all matching objects in the cache.
		prefix := path[0 : len(path)-1]
		m.lock.Lock()
		for key, item := range m.item {
			if strings.HasPrefix(key, prefix) {
				m.memoryUsage -= len(item)
				delete(m.item, key)
			}
		}
		m.lock.Unlock()
	} else {
		m.lock.Lock()
		m.memoryUsage -= len(m.item[path])
		delete(m.item, path)
		m.lock.Unlock()
	}

	if m.nextLevel != nil {
		m.nextLevel.Del(path)
	}
}

func (m *memoryCache) Get(path string) (item []byte) {
	m.lock.RLock()
	item = m.item[path]
	m.lock.RUnlock()

	if len(item) == 0 && m.backing != nil {
		item = m.backing.Get(path)
		if len(item) != 0 {
			m.set(path, item, false)
		}
	}
	return item
}

// Trim memory usage of the array. Right now this just clears all the data, which is obviously
// non-optimal. This function assumes that we already have a write lock.
func (m *memoryCache) trim() {
	m.item = make(map[string][]byte)
	m.memoryUsage = 0
}

// NewmemoryCache creates a new cache.
// 	memoryLimit is roughly the maximum amount of memory that will be used.
//  backing is an optional interface telling the cache where to look for more data on a miss.
//  nextLevel is an optional interface allowing write-through behavior.
func NewMemoryCache(memoryLimit int, backing Getter, nextLevel Cache) Cache {
	return &memoryCache{memoryLimit: memoryLimit,
		item:    make(map[string][]byte),
		backing: backing, nextLevel: nextLevel}
}
