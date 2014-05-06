package cache

import (
	"bytes"
	"compress/gzip"
	"strings"
	"time"
)

type Object struct {
	Data    []byte
	ModTime time.Time
}

type Filler interface {
	// Fill adds an object to the cache and also returns the object.
	// When the file may be compressed or not, Fill may add both versions to the cache, but should return
	// requested version of the item.
	Fill(cache Cache, path string) (Object, error)
}

type Cache interface {
	Get(path string, filler Filler) (Object, error)
	Set(path string, object Object, writeThrough bool) error
	// Delete an item from the cache. Include a "*" wildcard at the end to purge multiple items.
	Del(path string)
}

// Helper function for adding things to a Cache.
// It would be nice to have this as a member of Cache, but I need to figure out how to do that in Go.
func CompressAndSet(cache Cache, path string, data []byte, modTime time.Time) (uncompressed Object, compressed Object, err error) {

	compressedPath := path
	uncompressedPath := path

	if strings.HasSuffix(path, ".gz") {
		uncompressedPath = path[0 : len(path)-3]
	} else {
		compressedPath = path + ".gz"
	}

	gzBuf := new(bytes.Buffer)
	compressor, err := gzip.NewWriterLevel(gzBuf, gzip.BestCompression)
	if err != nil {
		return Object{}, Object{}, err
	}

	_, err = compressor.Write(data)
	compressor.Close()
	if err != nil {
		return Object{}, Object{}, err
	}

	compressedItem := Object{gzBuf.Bytes(), modTime}

	// Add the compressed version to the cache.
	err = cache.Set(compressedPath, compressedItem, true)
	if err != nil {
		return Object{}, Object{}, err
	}

	// Also add the uncompressed version.
	uncompressedItem := Object{data, modTime}
	err = cache.Set(uncompressedPath, uncompressedItem, true)

	return uncompressedItem, compressedItem, err
}
