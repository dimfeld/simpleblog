package cache

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
)

type DiskCache struct {
	lock     sync.RWMutex
	baseDir  string
	fileList map[string]int
}

func (d *DiskCache) Set(pathStr string, object Object) error {
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
	d.lock.Lock()
	d.fileList[pathStr] = len(object.Data)
	d.lock.Unlock()

	return nil
}

func (d *DiskCache) Del(pathStr string) {
	pathStr = path.Join(d.baseDir, pathStr)
	matches, err := filepath.Glob(pathStr)
	if err != nil {
		return
	}

	d.lock.Lock()
	for _, match := range matches {
		delete(d.fileList, match)
		err = os.RemoveAll(match)
		if err != nil {
			// log removal failure
		}
	}
	d.lock.Unlock()
}

func (d *DiskCache) Get(filename string, filler Filler) (Object, error) {
	cachePath := path.Join(d.baseDir, filename)
	d.lock.RLock()
	_, ok := d.fileList[cachePath]
	d.lock.RUnlock()
	if !ok {
		// The object is not currently present in the disk cache. Try to generate it.
		if filler != nil {
			return filler.Fill(d, filename)
		} else {
			return Object{}, errors.New(filename)
		}
	}

	f, err := os.Open(cachePath)
	if err != nil {
		// The object should be present, but is not. Try to generate it.
		if filler != nil {
			return filler.Fill(d, filename)
		} else {
			return Object{}, errors.New(filename)
		}
	}

	defer f.Close()

	fstat, err := f.Stat()
	if err != nil {
		return Object{}, err
	}
	modTime := fstat.ModTime()

	buf := bytes.Buffer{}
	buf.Grow(int(fstat.Size()))
	_, err = buf.ReadFrom(f)
	obj := Object{buf.Bytes(), modTime}

	return obj, err
}

func (d *DiskCache) initialScanWalkFunc(filename string, info os.FileInfo, err error) error {
	d.fileList[filename] = int(info.Size())
	return nil
}

func (d *DiskCache) ScanExisting() {
	filepath.Walk(d.baseDir, d.initialScanWalkFunc)
}

func NewDiskCache(baseDir string) *DiskCache {
	return &DiskCache{baseDir: baseDir, fileList: make(map[string]int)}
}
