package main

import (
	// "bufio"
	"bufio"
	"fmt"
	"github.com/dimfeld/gocache"
	"github.com/dimfeld/httppath"
	"github.com/dimfeld/httptreemux"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"
)

var (
	logger      *log.Logger
	debugLogger *log.Logger
	debugMode   bool
)

func debugf(format string, args ...interface{}) {
	if debugMode {
		debugLogger.Printf(format, args...)
	}
}

func debug(args ...interface{}) {
	if debugMode {
		debugLogger.Println(args...)
	}
}
func catchSIGINT(f func(), quit bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			f()
			if quit {
				os.Exit(1)
			}
		}
	}()
}

type GlobalData struct {
	*sync.RWMutex
	// Configuration Data
	indexPosts          int
	postsDir            string
	dataDir             http.Dir
	tagsPath            string
	tagsPageReverseSort bool

	// General cache
	cache    gocache.Cache
	memCache gocache.Cache
	archive  ArchiveSpecList
}

type simpleBlogHandler func(*GlobalData, http.ResponseWriter, *http.Request, map[string]string)

func handlerWrapper(handler simpleBlogHandler, globalData *GlobalData) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		logger.Printf("%s %s %s", r.Method, r.RequestURI)
		startTime := time.Now()
		handler(globalData, w, r, urlParams)
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		logger.Printf("   Handled in %d us", duration/time.Microsecond)
	}
}

func fileWrapper(filename string, handler httptreemux.HandlerFunc) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		urlParams["file"] = filename
		handler(w, r, urlParams)
	}
}

func filePrefixWrapper(prefix string, handler httptreemux.HandlerFunc) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		urlParams["file"] = "/" + filepath.Join(prefix, httppath.Clean(urlParams["file"]))
		handler(w, r, urlParams)
	}
}

func isDirectory(dirPath string) bool {
	stat, err := os.Stat(dirPath)
	if err != nil || !stat.IsDir() {
		return false
	}
	return true
}

func main() {
	// TODO Load these from configuration
	cacheDir, _ := filepath.Abs("cache")
	dataDirStr, _ := filepath.Abs("./testdata")
	dataDir := http.Dir(dataDirStr)
	postsDir := filepath.Join(dataDirStr, "posts")
	logFilename, _ := filepath.Abs("simpleblog.log")
	logPrefix := ""
	tagsPageReverseSort := true
	indexPosts := 15
	debugMode = true

	logFile, err := os.OpenFile(logFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		fmt.Println("Could not open log file", logFilename)
		os.Exit(1)
	}

	logBuffer := bufio.NewWriter(logFile)

	closer := func() {
		logger.Println("Shutting down...")
		logBuffer.Flush()
		logFile.Sync()
		logFile.Close()
	}

	catchSIGINT(closer, true)
	defer closer()

	var logWriter io.Writer = logBuffer
	if debugMode {
		// In debug mode, use unbuffered logging so that they come out right away.
		logWriter = logFile
	}

	logger = log.New(logWriter, logPrefix, log.LstdFlags)
	debugLogger = log.New(logWriter, "DEBUG ", log.LstdFlags)
	logger.Println("Starting...")

	diskCache, err := gocache.NewDiskCache(cacheDir)
	if err != nil {
		logger.Fatal("Could not create disk cache in", cacheDir)
	}

	if !isDirectory(dataDirStr) {
		logger.Fatal("Could not find data directory", dataDirStr)
	}

	if !isDirectory(postsDir) {
		logger.Fatal("Could not find posts directory", postsDir)
	}

	if !isDirectory(filepath.Join(dataDirStr, "assets")) {
		logger.Fatal("Could not find assets directory", filepath.Join(dataDirStr, "assets"))
	}

	if !isDirectory(filepath.Join(dataDirStr, "images")) {
		logger.Fatal("Could not find assets directory", filepath.Join(dataDirStr, "images"))
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

	tagsPath := filepath.Join(cacheDir, "tags.json")
	os.Remove(tagsPath)
	globalData := &GlobalData{
		RWMutex:             &sync.RWMutex{},
		cache:               multiLevelCache,
		memCache:            memCache,
		dataDir:             dataDir,
		postsDir:            postsDir,
		tagsPath:            tagsPath,
		tagsPageReverseSort: tagsPageReverseSort,
		indexPosts:          indexPosts,
	}

	archive, err := NewArchiveSpecList(postsDir)
	if err != nil {
		logger.Fatal("Could not create archive list: ", err)
	}
	globalData.archive = archive

	go watchFiles(globalData)

	router := httptreemux.New()
	router.PanicHandler = httptreemux.ShowErrorsPanicHandler

	router.GET("/", handlerWrapper(indexHandler, globalData))
	router.GET("/:year/:month/", handlerWrapper(archiveHandler, globalData))
	router.GET("/:year/:month/:post", handlerWrapper(postHandler, globalData))

	router.GET("/images/*file", filePrefixWrapper("images",
		handlerWrapper(staticNoCompressHandler, globalData)))
	router.GET("/assets/*file", filePrefixWrapper("assets",
		handlerWrapper(staticCompressHandler, globalData)))

	// No tags yet.
	router.GET("/tag/:tag", handlerWrapper(tagHandler, globalData))
	// No pagination yet.
	//router.GET("/tag/:tag/:page", handlerWrapper(tagHandler, globalData))

	router.GET("/:page", handlerWrapper(pageHandler, globalData))
	router.GET("/favicon.ico", fileWrapper("assets/favicon.ico",
		handlerWrapper(staticCompressHandler, globalData)))

	http.ListenAndServe(":8080", router)
}
