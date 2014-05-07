package main

import (
	cachePkg "github.com/dimfeld/simpleblog/cache"
	"github.com/julienschmidt/httprouter"

	//	"io/ioutil"
	"net/http"
)

type Article struct {
	Title   string
	Tags    []string
	Content []byte
}

type GlobalData struct {
	// General cache
	cache cachePkg.Cache
}

type simpleBlogHandler func(*GlobalData, http.ResponseWriter, *http.Request, map[string]string)

func handlerWrapper(handler simpleBlogHandler, globalData *GlobalData) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		handler(globalData, w, r, urlParams)
	}
}

func main() {
	// TODO Load configuration

	diskCache := cachePkg.NewDiskCache("cache")
	// Large memory cache uses 64 MiB at most, with the larges object being 8 MiB.
	largeObjectLimit := 8 * 1024 * 1024
	largeMemCache := cachePkg.NewMemoryCache(64*1024*1024, largeObjectLimit)
	// Small memory cache uses 16 MiB at most, with the largest object being 16KiB.
	smallObjectLimit := 8 * 1024 * 1024
	smallMemCache := cachePkg.NewMemoryCache(16*1024*1024, smallObjectLimit)

	memCache := cachePkg.NewSplitSize(
		cachePkg.SplitSizeChild{smallObjectLimit, smallMemCache},
		cachePkg.SplitSizeChild{largeObjectLimit, largeMemCache})

	multiLevelCache := cachePkg.MultiLevel{0: memCache, 1: diskCache}

	globalData := &GlobalData{cache: multiLevelCache}

	router := httprouter.New()
	router.GET("/", handlerWrapper(indexHandler, globalData))
	router.GET("/:year/:month", handlerWrapper(archiveHandler, globalData))
	router.GET("/:year/:month/:post", handlerWrapper(postHandler, globalData))
	// No tags yet.
	//router.GET("/tag/:tag", handlerWrapper(tagHandler, globalData))
	// No pagination yet.
	//router.GET("/tag/:tag/:page", handlerWrapper(tagHandler, globalData))
	router.GET("/images/*file", handlerWrapper(simpleHandler, globalData))
	router.GET("/assets/*file", handlerWrapper(staticCompressHandler, globalData))
	router.GET("/favicon.ico", handlerWrapper(staticCompressHandler, globalData))

	http.ListenAndServe(":8080", router)
}
