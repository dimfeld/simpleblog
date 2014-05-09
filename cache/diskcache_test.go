package cache

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func makeDiskCache(t testing.TB) *DiskCache {
	dir, err := ioutil.TempDir("", "diskcache-test")
	if err != nil {
		t.Errorf("Could not create temporary directory: %s", err)
		t.FailNow()
	}
	return NewDiskCache(dir)
}

func cleanup(c *DiskCache) {
	os.RemoveAll(c.baseDir)
}

// TestDiskCacheLocation ensures that the disk cache's files are actually placed under the cache root.
func TestDiskCacheLocation(t *testing.T) {
	c := makeDiskCache(t)
	defer cleanup(c)

	c.Set("abc", simpleObject(8))
	c.Set("subdir/abc", simpleObject(8))

	_, err := os.Stat(path.Join(c.baseDir, "abc"))
	if err != nil {
		t.Error("Couldn't find file abc:", err)
	}

	stat, err := os.Stat(path.Join(c.baseDir, "subdir"))
	if err != nil {
		t.Error("Couldn't find directory subdir:", err)
	} else if !stat.IsDir() {
		t.Error("subdir is not a directory")
	}

	_, err = os.Stat(path.Join(c.baseDir, "subdir", "abc"))
	if err != nil {
		t.Error("Couldn't find file subdir/abc:", err)
	}
}

func TestDiskCacheBasic(t *testing.T) {
	c := makeDiskCache(t)
	defer cleanup(c)
	basicCacheTest(t, c, 16)
}

func TestDiskCacheWildcardDeletes(t *testing.T) {
	c := makeDiskCache(t)
	defer cleanup(c)
	testWildcardDelete(t, c)
}

func TestDiskCacheFiller(t *testing.T) {
	c := makeDiskCache(t)
	defer cleanup(c)
	testCacheFiller(t, c)
}

func TestDiskCacheParallelSets(t *testing.T) {
	c := makeDiskCache(t)
	defer cleanup(c)
	testParallelSets(t, c, 1000, 16, 10, true)
}

func BenchmarkDiskCacheSingle(b *testing.B) {
	c := makeDiskCache(b)
	defer cleanup(c)
	benchmarkSingleCache(b, c, 16, false)
}

func BenchmarkDiskCacheGet(b *testing.B) {
	c := makeDiskCache(b)
	defer cleanup(c)
	benchmarkSingleCacheGet(b, c, 16, false)
}

func BenchmarkDiskCacheSet(b *testing.B) {
	c := makeDiskCache(b)
	defer cleanup(c)
	benchmarkSingleCacheSet(b, c, 16, false)
}

func BenchmarkDiskCacheParallelSets(b *testing.B) {
	c := makeDiskCache(b)
	defer cleanup(c)
	benchmarkParallelSets(b, c, 16, 10)
}
