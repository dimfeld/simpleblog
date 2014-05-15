package main

import (
	"bufio"
	"fmt"
	"github.com/dimfeld/gocache"
	"github.com/dimfeld/httptreemux"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	logger *log.Logger
)

type GlobalData struct {
	// Configuration Data
	indexPosts int
	postsDir   string
	dataDir    http.Dir
	tagsPath   string

	// General cache
	cache    gocache.Cache
	memCache gocache.Cache
	archive  ArchiveSpecList
}

type simpleBlogHandler func(*GlobalData, http.ResponseWriter, *http.Request, map[string]string)

func handlerWrapper(handler simpleBlogHandler, globalData *GlobalData) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		handler(globalData, w, r, urlParams)
	}
}

func fileWrapper(handler httptreemux.HandlerFunc, filename string) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		urlParams["file"] = filename
		handler(w, r, urlParams)
	}
}

func main() {
	// TODO Load these from configuration
	cacheDir, _ := filepath.Abs("cache")
	dataDirStr, _ := filepath.Abs("data")
	dataDir := http.Dir(dataDirStr)
	postsDir, _ := filepath.Abs("posts")
	logFilename, _ := filepath.Abs("simpleblog.log")
	logPrefix := "SimpleBlog"

	logFile, err := os.OpenFile(logFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		fmt.Println("Could not open log file", logFilename)
		os.Exit(1)
	}
	// Overkill?
	logBuffer := bufio.NewWriter(logFile)

	defer func() {
		logBuffer.Flush()
		logFile.Close()
	}()

	logger = log.New(logBuffer, logPrefix, log.LstdFlags)

	diskCache, err := gocache.NewDiskCache(cacheDir)
	if err != nil {
		panic(err)
	}

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

	go watchFiles(globalData)

	router := httptreemux.New()

	defer func() {
		if err := recover(); err != nil {
			router.Dump()
			panic(err)
		}
	}()

	router.GET("/", handlerWrapper(indexHandler, globalData))
	router.GET("/:year/:month", handlerWrapper(archiveHandler, globalData))
	router.GET("/:year/:month/:post", handlerWrapper(postHandler, globalData))

	router.GET("/images/*file", handlerWrapper(staticNoCompressHandler, globalData))
	router.GET("/assets/*file", handlerWrapper(staticCompressHandler, globalData))

	// No tags yet.
	router.GET("/tag/:tag", handlerWrapper(tagHandler, globalData))
	// No pagination yet.
	router.GET("/tag/:tag/:page", handlerWrapper(tagHandler, globalData))

	router.GET("/:page", handlerWrapper(pageHandler, globalData))
	router.GET("/favicon.ico", fileWrapper(
		handlerWrapper(staticCompressHandler, globalData), "favicon.ico"))

	http.ListenAndServe(":8080", router)
}
