package cache

import "testing"

func TestSplitCache(t *testing.T) {
	caches := []Cache{make(DummyCache), make(DummyCache)}
	threshold := 8
	// Add them out of order to make sure the sorting works.
	c := NewSplitSize(SplitSizeChild{0, caches[1]}, SplitSizeChild{threshold, caches[0]})

	if c[0].MaxSize != threshold && c[1].MaxSize != 0 {
		t.Error("Cache sort did not work properly")
	}

	checkObject := func(path string) {
		o, err := c.Get(path, nil)
		if err != nil {
			t.Errorf("Error retrieving object %s: %s", path, err)
		}
		size := len(o.Data)

		presentIndex := 0
		notPresentIndex := 1
		if size > threshold {
			presentIndex = 1
			notPresentIndex = 0
		}

		_, err = caches[presentIndex].Get(path, nil)
		if err != nil {
			t.Errorf("Object of size %d was not found in cache %d", size, presentIndex)
		}

		_, err = caches[notPresentIndex].Get(path, nil)
		if err == nil {
			t.Errorf("Object of size %d was found in cache %d", size, notPresentIndex)
		}
	}

	testDelete := func(path string) {
		c.Del(path)
		_, err := c.Get(path, nil)
		if err == nil {
			t.Error("Object %s still in cache after delete", path)
		}

		for i, item := range c {
			_, err = item.Cache.Get(path, nil)
			if err == nil {
				t.Error("Object %s still in cache %d after delete", i, path)
			}
		}
	}

	testObject := func(path string, size int) {
		o := simpleObject(size)
		err := c.Set(path, o)
		if err != nil {
			t.Errorf("Error adding object %s of size %d: %s", path, size, err)
		}
		checkObject(path)
	}

	testObject("abc", threshold-1)
	testObject("abd", threshold)
	testObject("abe", threshold+1)

	testDelete("abc")
	testDelete("abe")

	f := dummyFiller{}
	_, err := c.Get("abc", f)
	if err != nil {
		t.Error("Small Get with Filler failed")
	}
	checkObject("abc")

	largePath := "abcdefghijklmnop"
	_, err = c.Get(largePath, f)
	if err != nil {
		t.Errorf("Large Get with Filler failed")
	}
	checkObject(largePath)

	c = c.AddChildCache(100, make(DummyCache))
	if c[0].MaxSize != threshold || c[1].MaxSize != 100 || c[2].MaxSize != 0 {
		t.Errorf("AddChildCache did not sort correctly. Expected (%d, 100, 0), saw (%d, %d, %d)",
			threshold, c[0].MaxSize, c[1].MaxSize, c[2].MaxSize)
	}
}
