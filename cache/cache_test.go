package cache

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type DummyCache map[string]Object

func (d DummyCache) Keys() string {
	keyList := make([]string, 0)
	for key, _ := range d {
		keyList = append(keyList, key)
	}

	return strings.Join(keyList, ", ")
}

func (d DummyCache) Get(path string, filler Filler) (Object, error) {
	obj, ok := d[path]
	if !ok {
		return obj, errors.New("Item not found")
	}

	return obj, nil
}

func (d DummyCache) Set(path string, obj Object) error {
	d[path] = obj
	return nil
}

func (d DummyCache) Del(path string) {
	delete(d, path)
}

func unzip(compressed []byte) ([]byte, error) {
	compressedBuf := bytes.NewReader(compressed)
	ungz, err := gzip.NewReader(compressedBuf)
	if err != nil {
		return nil, err
	}
	unzipped := bytes.Buffer{}
	_, err = unzipped.ReadFrom(ungz)
	return unzipped.Bytes(), err
}

func Equal(one, two Object) bool {
	// Truncate because disk caches don't store subsecond data.
	oneModTime := one.ModTime.Truncate(time.Second)
	twoModTime := two.ModTime.Truncate(time.Second)
	return bytes.Equal(one.Data, two.Data) && oneModTime == twoModTime
}

func (o Object) String() string {
	return fmt.Sprintf("{%s, %s}", string(o.Data), o.ModTime.String())
}

func simpleObject(length int) Object {
	return Object{make([]byte, length), time.Now()}
}

func TestTestFunctions(t *testing.T) {
	o := Object{[]byte("abc/def"), time.Now()}
	if !Equal(o, o) {
		t.Error("Equal function is broken")
	}
}

func TestCompressAndSet(t *testing.T) {
	c := make(DummyCache)

	key := "abc/def"
	data := []byte("abcdef")
	modTime := time.Now()

	baseObj := Object{data, modTime}

	uncompressed, compressed, err := CompressAndSet(c, key, data, modTime)
	if err != nil {
		t.Errorf("CompressAndSet returned error %s", err)
	}

	if !Equal(uncompressed, baseObj) {
		t.Errorf("Returned uncompressed object \"%s\" not equal to original \"%s\"",
			uncompressed.String(), baseObj.String())
	}

	unzipped, err := unzip(compressed.Data)
	if err != nil {
		t.Errorf("Error decompressing returned data: %s", err)
	}

	unzippedObj := Object{unzipped, compressed.ModTime}
	if !Equal(unzippedObj, baseObj) {
		t.Errorf("Returned compressed object \"%s\" not equal to original \"%s\"",
			unzippedObj.String(), baseObj.String())
	}

	cacheUncompressed, err := c.Get(key, nil)
	if err != nil {
		t.Errorf("Key %s not in cache. Cache contains %s", key, c.Keys())
	}
	if !Equal(cacheUncompressed, baseObj) {
		t.Errorf("Cached uncompressed object \"%s\" not equal to original \"%s\"",
			cacheUncompressed.String(), baseObj.String())
	}

	cacheCompressed, err := c.Get(key+".gz", nil)
	if err != nil {
		t.Errorf("Key %s not in cache. Cache contains %s", key+".gz", c.Keys())
	}
	unzipped, err = unzip(cacheCompressed.Data)
	if err != nil {
		t.Errorf("Error decompressing cached data: %s", err)
	}
	cacheUnzipped := Object{unzipped, cacheCompressed.ModTime}
	if !Equal(cacheUnzipped, baseObj) {
		t.Errorf("Cached compressed object \"%s\" not equal to original \"%s\"",
			cacheUnzipped.String(), baseObj.String())
	}
}

func generatePaths(prefix string, numPaths int) []string {
	paths := make([]string, numPaths)
	for i, _ := range paths {
		paths[i] = fmt.Sprintf("%s%d/%d", prefix, i/10, i)
	}
	return paths
}

func basicCacheTest(t *testing.T, c Cache, objectSize int) {
	obj := simpleObject(objectSize)
	err := c.Set("abc", obj)
	if err != nil {
		t.Errorf("Error adding object: %s", err)
	}
	ret, err := c.Get("abc", nil)
	if err != nil {
		t.Errorf("Error getting object: %s", err)
	}
	if !Equal(obj, ret) {
		t.Errorf("Cache returned %s, expected %s", ret.String(), obj.String())
	}

	c.Del("abc")
	_, err = c.Get("abc", nil)
	if err == nil {
		t.Errorf("Object still in cache after delete")
	}
}

func singleCacheTest(t testing.TB, c Cache, paths []string, objectSize int, allowLoss bool) {
	o := simpleObject(objectSize)

	for _, p := range paths {
		c.Set(p, o)
	}

	for _, p := range paths {
		_, err := c.Get(p, nil)
		if !allowLoss && err != nil {
			t.Errorf("Failed to retrieve path %s: %s.", p, err.Error())
		}
	}
}

func benchmarkSingleCache(b *testing.B, c Cache, objectSize int, allowLoss bool) {
	paths := generatePaths("", b.N)

	b.ResetTimer()
	singleCacheTest(b, c, paths, objectSize, allowLoss)
}

func benchmarkSingleCacheGet(b *testing.B, c Cache, objectSize int, allowLoss bool) {
	o := simpleObject(objectSize)
	paths := generatePaths("", b.N)

	for _, p := range paths {
		c.Set(p, o)
	}

	b.ResetTimer()
	for _, p := range paths {
		_, err := c.Get(p, nil)
		if !allowLoss && err != nil {
			b.Errorf("Failed to retrieve path %s: %s.", p, err.Error())
		}
	}
}

func benchmarkSingleCacheSet(b *testing.B, c Cache, objectSize int, allowLoss bool) {
	o := simpleObject(objectSize)
	paths := generatePaths("", b.N)

	b.ResetTimer()
	for _, p := range paths {
		c.Set(p, o)
	}
}

func parallelSetsLoop(t testing.TB, c Cache, paths []string, objectSize int, filler Filler,
	wg *sync.WaitGroup, start *sync.RWMutex, verify bool) {
	// Wait until the creator unlocks the mutex.
	start.RLock()
	data := []byte(strconv.Itoa(rand.Int()))
	o := Object{data, time.Now()}

	for _, p := range paths {
		c.Set(p, o)
	}

	if verify {
		for _, p := range paths {
			obj, err := c.Get(p, filler)
			if err != nil {
				t.Error("Error retrieving", p, err)
			} else if !Equal(obj, o) {
				throwError := true
				if filler != nil {
					filledObj, _ := filler.Fill(nil, p)
					if Equal(obj, filledObj) {
						throwError = false
					}
				}

				if throwError {
					t.Errorf("Object at path %s was %s, expected %s", p, obj.String(), o.String())
				}
			}
		}
	}

	start.RUnlock()
	wg.Done()
}

func testParallelSets(t testing.TB, c Cache, iterations int, objectSize int, filler Filler,
	numGoroutines int, verify bool) {
	rand.Seed(1)
	wg := &sync.WaitGroup{}
	start := &sync.RWMutex{}
	start.Lock()

	for i := 0; i < numGoroutines; i++ {
		paths := generatePaths(strconv.Itoa(i), iterations)
		wg.Add(1)
		go parallelSetsLoop(t, c, paths, objectSize, filler, wg, start, verify)
	}

	b, ok := t.(*testing.B)
	if ok {
		b.ResetTimer()
	}

	// Unlock the mutex to start all the goroutines running.
	start.Unlock()
	wg.Wait()
}

func benchmarkParallelSets(b *testing.B, c Cache, objectSize int, numGoroutines int) {
	testParallelSets(b, c, b.N/numGoroutines, objectSize, nil, numGoroutines, false)
}

type dummyFiller struct {
	prefix  string
	modTime time.Time
	err     error
}

func (d dummyFiller) Fill(cache Cache, path string) (Object, error) {
	if d.err != nil {
		return Object{}, d.err
	}

	modTime := d.modTime
	if modTime.IsZero() {
		modTime = time.Now()
	}

	data := strings.Join([]string{d.prefix, path}, "-")
	obj := Object{[]byte(data), d.modTime}
	if cache != nil {
		cache.Set(path, obj)
	}
	return obj, nil
}

func testCacheFiller(t *testing.T, c Cache) {
	f := dummyFiller{prefix: "fill", modTime: time.Now()}

	_, err := c.Get("abc", f)
	if err != nil {
		t.Error("Get with Filler returned error", err)
	}

	_, err = c.Get("abc", nil)
	if err != nil {
		t.Error("Get with Filler did not actually fill cache")
	}

	obj := Object{[]byte("test object"), time.Now()}
	err = c.Set("abc", obj)
	if err != nil {
		t.Error("Failed to overwrite object after Filler-backed get")
	}

	obj2, err := c.Get("abc", f)
	if !Equal(obj, obj2) {
		t.Error("Get with Filler should have had cache hit and not used Filler")
	}
}

func testWildcardDelete(t *testing.T, c Cache) {
	paths := []string{"abc", "abd", "subdir/abc", "subdir/abd", "ddd", "subdir/ddd"}

	testPattern := func(pattern string) {
		expected := simpleObject(8)
		for _, p := range paths {
			c.Set(p, expected)
		}

		wildcard := pattern + "*"
		t.Log("Testing delete of", wildcard)
		c.Del(wildcard)
		for _, p := range paths {
			o, err := c.Get(p, nil)
			if strings.HasPrefix(p, pattern) {
				if err == nil {
					t.Errorf("Delete of %s should delete %s", wildcard, p)
				}
			} else {
				if err != nil {
					t.Errorf("Delete of %s should not delete %s", wildcard, p)
				}

				if !Equal(expected, o) {
					t.Errorf("Path %s contained %s, expected %s",
						p, o.String(), expected.String())

				}
			}

		}
	}

	testPattern("abc")
	testPattern("ab")
	testPattern("subdir")
	testPattern("subdir/a")
	testPattern("subdir/abd")
	testPattern("d")
}
