package cache

import (
	"os"
	"path/filepath"
)

type DiskCache struct {
}

func (d *DiskCache) Add(path string, object []byte) {

}

func (d *DiskCache) Del(path string) {
	path = "cache/" + path
	matches, err := filepath.Glob(path)

	if err != nil {
		return
	}

	for i := range matches {
		err = os.RemoveAll(matches[i])
		if err != nil {
			// Syslog removal failure
		}
	}
}

func (d *DiskCache) Get(path string) {

}
