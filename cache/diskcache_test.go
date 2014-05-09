package cache

import (
	"io/ioutil"
	"os"
	"testing"
)

func makeDiskCache(t *testing.T) *DiskCache {
	dir, err := ioutil.TempDir("", "diskcache-test")
	t.Log("Using temporary directory", dir)
	if err != nil {
		t.Errorf("Could not create temporary directory: %s", err)
		t.FailNow()
	}
	return NewDiskCache(dir)
}

func cleanup(c *DiskCache) {
	os.RemoveAll(c.baseDir)
}

func TestDiskCacheBasic(t *testing.T) {
	c := makeDiskCache(t)
	defer cleanup(c)
	basicCacheTest(t, c, 16)

}
