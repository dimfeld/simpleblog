package cache

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type DiskCache struct {
	baseDir string
}

func (d *DiskCache) Set(pathStr string, object Object, writeThrough bool) error {
	// Ignore writeThrough since there's no backing cache here.
	pathStr = path.Join(d.baseDir, pathStr)
	dirPath, _ := path.Split(pathStr)

	err := os.MkdirAll(dirPath, 0700)
	if err != nil {
		return fmt.Errorf("Could not create directory %s: %s", dirPath, err.Error())
	}

	err = ioutil.WriteFile(pathStr, object.Data, 0600)
	if err != nil {
		return fmt.Errorf("Could not write file %s: %s", pathStr, err.Error())
	}

	os.Chtimes(pathStr, object.ModTime, object.ModTime)

	return nil
}

func (d *DiskCache) Del(pathStr string) {
	pathStr = path.Join(d.baseDir, pathStr)
	matches, err := filepath.Glob(pathStr)
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

func (d *DiskCache) Get(filename string, filler Filler) (Object, error) {
	cachePath := d.baseDir + filename
	f, err := os.Open(cachePath)
	if err != nil {
		// The object is not currently present in the disk cache. Try to generate it.
		return filler.Fill(d, filename)
	}

	defer f.Close()

	fstat, err := f.Stat()
	if err != nil {
		return Object{}, err
	}
	modTime := fstat.ModTime()

	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(f)
	obj := Object{buf.Bytes(), modTime}

	return obj, err
}

func NewDiskCache(baseDir string) Cache {
	return &DiskCache{baseDir}
}
