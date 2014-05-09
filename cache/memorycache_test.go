package cache

import (
	"strconv"
	"testing"
)

func TestMemoryCacheBasic(t *testing.T) {
	c := NewMemoryCache(16*1024, 0)
	basicCacheTest(t, c, 8)
}

func TestMemoryCacheObjectSizeLimit(t *testing.T) {
	limit := 8
	c := NewMemoryCache(16*1024, limit)
	c.Set("abc", simpleObject(limit+1))
	_, err := c.Get("abc", nil)
	if err == nil {
		t.Error("Cache did not reject too-large object")
	}

	c.Set("abc", simpleObject(limit))
	_, err = c.Get("abc", nil)
	if err != nil {
		t.Error("Cache rejected object at size limit")
	}

	c.Set("abc", simpleObject(limit+1))
	o, err := c.Get("abc", nil)
	if err == nil {
		if len(o.Data) == limit {
			t.Error("Replacing object with too-large object did nothing")
		} else if len(o.Data) == limit+1 {
			t.Error("Cache did not reject too-large object replacing another object")
		} else {
			t.Error("Something weird happened. Cache returned %s", o.String())
		}
	}
}

func TestMemoryCacheTotalSizeLimit(t *testing.T) {
	limit := 1024
	c := NewMemoryCache(limit, 0)
	if c.memoryLimit != limit {
		t.Errorf("Cache memory limit is %d, wanted %d", c.memoryLimit, limit)
	}
	objectSize := 100
	objects := limit/objectSize + 1
	o := simpleObject(objectSize)

	for i := 0; i < objects; i++ {
		c.Set(strconv.Itoa(i), o)
	}

	if c.memoryUsage > limit {
		t.Errorf("Cache memory usage is %d, limit is %d", c.memoryUsage, limit)
	}

	missedOne := false
	for i := 0; i < objects; i++ {
		_, err := c.Get(strconv.Itoa(i), nil)
		if err != nil {
			missedOne = true
			break
		}
	}

	if !missedOne {
		t.Error("Cache above memory limit did not trim any objects")
	}
}

func TestMemoryCacheParallelSets(t *testing.T) {
	c := NewMemoryCache(1024*1024*1024, 0)
	testParallelSets(t, c, 1000, 16, 10, true)
}

func TestMemoryCacheWildcardDeletes(t *testing.T) {
	c := NewMemoryCache(1024*1024, 0)
	testWildcardDelete(t, c)
}

func BenchmarkMemoryCacheSingle(b *testing.B) {
	c := NewMemoryCache(b.N*32, 0)
	benchmarkSingleCache(b, c, 16, false)
}

func BenchmarkMemoryCacheGet(b *testing.B) {
	c := NewMemoryCache(b.N*32, 0)
	benchmarkSingleCacheGet(b, c, 16, false)
}

func BenchmarkMemoryCacheSet(b *testing.B) {
	c := NewMemoryCache(b.N*32, 0)
	benchmarkSingleCacheSet(b, c, 16, false)
}

func BenchmarkMemoryCacheParallelSets(b *testing.B) {
	c := NewMemoryCache(1024*1024*1024, 0)
	benchmarkParallelSets(b, c, 16, 10)
}

// BenchmarkMemoryCacheWithTrim tests the performance of Parallel sets
// when the sets exceed the memory limit, causing trim operations.
func BenchmarkMemoryCacheWithTrim(b *testing.B) {
	c := NewMemoryCache(1024, 0)
	benchmarkParallelSets(b, c, 250, 10)
}

func BenchmarkWildcardDeletes(b *testing.B) {

}
