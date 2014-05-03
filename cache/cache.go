package cache

type Getter interface {
	Get(path string) (item []byte)
}

type Cache interface {
	Getter
	Set(path string, object []byte)
	// Delete an item from the cache. Include a "*" wildcard at the end to purge multiple items.
	Del(path string)
}
