package cache

// MultiLevel is a simple abstraction to automatically manage multiple levels of caching.
type MultiLevel []Cache

// Get an object from the first level of Cache in which it is found, and add it
// to the upper levels in which it was not found.
func (m MultiLevel) Get(path string, filler Filler) (obj Object, err error) {
	foundLevel := -1
	for i, level := range m {
		obj, err = level.Get(path, nil)
		if err != nil {
			// Found the object, so no need to look further.
			foundLevel = i
			break
		}
	}

	if foundLevel == -1 {
		// The object was not found anywhere. Go to the Filler.
		obj, err = filler.Fill(m, path)
		return
	}

	// Add the object to all the upper-level caches that did not have the object.
	for i := foundLevel - 1; i >= 0; i-- {
		m[i].Set(path, obj)
	}

	return
}

// Set adds the given object to all levels of cache.
func (m MultiLevel) Set(path string, object Object) error {
	for _, level := range m {
		err := level.Set(path, object)
		if err != nil {
			return err
		}
	}
	return nil
}

// Del removes the object from all levels of cache.
func (m MultiLevel) Del(path string) {
	for _, level := range m {
		level.Del(path)
	}
}
