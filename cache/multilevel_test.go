package cache

import (
	"errors"
	"testing"
)

func TestMultiLevelCache(t *testing.T) {
	c := MultiLevel{0: make(DummyCache), 1: make(DummyCache), 2: make(DummyCache)}

	checkInternal := func(p string, failDesc string) {
		for i := 0; i <= 2; i++ {
			_, err := c[i].Get(p, nil)
			if err != nil {
				t.Error(failDesc)
				t.Errorf("Cache level %d does not have %s", i, p)
			}
		}
	}

	o := simpleObject(8)

	c.Set("abc", o)
	checkInternal("abc", "Item was not added to all levels")

	c[1].Del("abc")
	c[2].Del("abc")
	_, err := c.Get("abc", nil)
	if err != nil {
		t.Error("Purging lower levels caused get failure")
	}

	_, err = c[1].Get("abc", nil)
	if err == nil {
		t.Error("Get filled in lower levels of cache")
	}

	c.Set("abc", o)
	checkInternal("abc", "Set with already existing item did not fill in lower levels")

	c[0].Del("abc")
	_, err = c.Get("abc", nil)
	if err != nil {
		t.Error("Get failed after first level cache purged object")
	}
	checkInternal("abc", "Get did not fill in inner cache")

	c.Del("abc")
	_, err = c.Get("abc", nil)
	if err == nil {
		t.Error("Get succeeded after delete")
	}

	for i := 0; i < 3; i++ {
		_, err = c[i].Get("abc", nil)
		if err == nil {
			t.Errorf("Level %d cache has data after delete", i)
		}
	}

	filler := dummyFiller{}
	_, err = c.Get("abc", filler)
	if err != nil {
		t.Error("Get with Filler returned error")
	}
	checkInternal("abc", "Filler did not set all levels of cache")

	filler.err = errors.New("Test Filler Error")
	_, err = c.Get("aaa", filler)
	if err == nil {
		t.Error("Get with Filler error did not return error")
	}
}
