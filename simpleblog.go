package main

import (
	cachePkg "github.com/dimfeld/simpleblog/cache"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"path/filepath"
)

type GlobalData struct {
	// General cache
	cache    cachePkg.Cache
	memCache cachePkg.Cache
	logger   log.Logger
	postsDir string
	dataDir  http.Dir
	tagsPath string
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

	diskCache := cachePkg.NewDiskCache(cacheDir)
	diskCache.ScanExisting()

	// Large memory cache uses 64 MiB at most, with the largest object being 8 MiB.
	largeObjectLimit := 8 * 1024 * 1024
	largeMemCache := cachePkg.NewMemoryCache(64*1024*1024, largeObjectLimit)
	// Small memory cache uses 16 MiB at most, with the largest object being 16KiB.
	smallObjectLimit := 16 * 1024
	smallMemCache := cachePkg.NewMemoryCache(16*1024*1024, smallObjectLimit)

	// Create a split cache, putting all objects smaller than 16 KiB into the small cache.
	// This split cache prevents a few large objects from evicting all the smaller objects.
	memCache := cachePkg.NewSplitSize(
		cachePkg.SplitSizeChild{smallObjectLimit, smallMemCache},
		cachePkg.SplitSizeChild{largeObjectLimit, largeMemCache})

	multiLevelCache := cachePkg.MultiLevel{0: memCache, 1: diskCache}

	globalData := &GlobalData{
		cache:    multiLevelCache,
		memCache: memCache,
		dataDir:  dataDir,
		postsDir: postsDir,
		tagsPath: filepath.Join(cacheDir, "tags.json"),
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
