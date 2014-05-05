package main

import (
	"github.com/dimfeld/simpleblog/cache"
	"strings"
	"time"
	//"io/ioutil"
	//"html/template"
)

type PageGenerator func(map[string]string) ([]byte, error)

type PageSpec struct {
	generator PageGenerator
	params    map[string]string
}

func (ps PageSpec) Fill(cacheObj cache.Cache, path string) (cache.Object, error) {
	data, err := ps.generator(ps.params)
	if err != nil {
		return cache.Object{}, err
	}

	uncompressed, compressed, err := cache.CompressAndSet(cacheObj, path, data, time.Now())
	if strings.HasSuffix(path, ".gz") {
		return compressed, err
	} else {
		return uncompressed, err
	}
}

func generatePostPage(params map[string]string) ([]byte, error) {
	return nil, nil
}

func generateArchivePage(params map[string]string) ([]byte, error) {
	return nil, nil
}

func generateTagsPage(params map[string]string) ([]byte, error) {
	return nil, nil
}

func generateSinglePost(path string) ([]byte, error) {
	return nil, nil
}
