package main

import (
	"github.com/dimfeld/gocache"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"path/filepath"
)

type GlobalData struct {
	// General cache
	cache      gocache.Cache
	memCache   gocache.Cache
	logger     log.Logger
	postsDir   string
	dataDir    http.Dir
	tagsPath   string
	indexPosts int
}

type simpleBlogHandler func(*GlobalData, http.ResponseWriter, *http.Request, map[string]string)

func handlerWrapper(handler simpleBlogHandler, globalData *GlobalData) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		handler(globalData, w, r, urlParams)
	}
}

func main() {
	// TODO Load these from configuration
	cacheDir := "./cache"
	dataDir := http.Dir("./data")
	postsDir := "./posts"

	diskCache := gocache.NewDiskCache(cacheDir)
	diskCache.ScanExisting()

	// Large memory cache uses 64 MiB at most, with the largest object being 8 MiB.
	largeObjectLimit := 8 * 1024 * 1024
	largeMemCache := gocache.NewMemoryCache(64*1024*1024, largeObjectLimit)
	// Small memory cache uses 16 MiB at most, with the largest object being 16KiB.
	smallObjectLimit := 16 * 1024
	smallMemCache := gocache.NewMemoryCache(16*1024*1024, smallObjectLimit)

	// Create a split cache, putting all objects smaller than 16 KiB into the small cache.
	// This split cache prevents a few large objects from evicting all the smaller objects.
	memCache := gocache.NewSplitSize(
		gocache.SplitSizeChild{MaxSize: smallObjectLimit, Cache: smallMemCache},
		gocache.SplitSizeChild{MaxSize: largeObjectLimit, Cache: largeMemCache})

	multiLevelCache := gocache.MultiLevel{0: memCache, 1: diskCache}

	globalData := &GlobalData{
		cache:      multiLevelCache,
		memCache:   memCache,
		dataDir:    dataDir,
		postsDir:   postsDir,
		tagsPath:   filepath.Join(cacheDir, "tags.json"),
		indexPosts: 15,
	}

	watchFiles(globalData)

	router := httprouter.New()
	router.GET("/", handlerWrapper(indexHandler, globalData))
	router.GET("/:year/:month", handlerWrapper(archiveHandler, globalData))
	router.GET("/:year/:month/:post", handlerWrapper(postHandler, globalData))
	// No tags yet.
	//router.GET("/tag/:tag", handlerWrapper(tagHandler, globalData))
	// No pagination yet.
	//router.GET("/tag/:tag/:page", handlerWrapper(tagHandler, globalData))
	router.GET("/images/*file", handlerWrapper(staticNoCompressHandler, globalData))
	router.GET("/assets/*file", handlerWrapper(staticCompressHandler, globalData))
	router.GET("/favicon.ico", handlerWrapper(staticCompressHandler, globalData))

	http.ListenAndServe(":8080", router)
}
