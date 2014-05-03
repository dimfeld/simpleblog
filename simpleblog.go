package main

import (
	cachePkg "github.com/dimfeld/simpleblog/cache"
	//	"io/ioutil"
	"net/http"
)

type Article struct {
	Title   string
	Tags    []string
	Content []byte
}

type GlobalData struct {
	cache cachePkg.Cache
}

func noCacheHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	if !exists(path) {
		http.Error(w, "File not found", http.StatusNotFound)
	}
	http.ServeFile(w, r, path)
}

func main() {
	// TODO Load configuration

	memCache := cachePkg.NewMemoryCache(64*1024*1024, nil, nil)

	globalData := &GlobalData{cache: memCache}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		viewHandler(globalData, w, r)
	})
	http.HandleFunc("/images/", noCacheHandler)
	http.ListenAndServe(":8080", nil)
}
