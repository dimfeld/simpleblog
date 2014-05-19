package main

import (
	// "bufio"
	"bufio"
	"fmt"
	"github.com/dimfeld/gocache"
	"github.com/dimfeld/goconfig"
	"github.com/dimfeld/httppath"
	"github.com/dimfeld/httptreemux"
	"html/template"
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
	config      *Config
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
			logger.Println("SIGINT received...")
			f()
			if quit {
				os.Exit(1)
			}
		}
	}()
}

type GlobalData struct {
	*sync.RWMutex

	// General cache
	cache    gocache.Cache
	memCache gocache.Cache

	archive   ArchiveSpecList
	templates *template.Template
}

type Config struct {
	// Number of posts to display on the main page.
	IndexPosts int
	// True if /tag/<tag> should sort posts in descending order.
	TagsPageNewestFirst bool
	// True if archive list at the bottom should start with the latest month.
	ArchiveListNewestFirst bool

	// Directory to search for posts.
	PostsDir string
	// Directory to search for static data.
	DataDir string
	// Directory to use for the disk cache.
	CacheDir string
	// File path to store tags.json.
	TagsPath string

	LogFile   string
	LogPrefix string
	// Flush the log every X seconds.
	LogFlushPeriod int

	Domain string
	Port   int

	LargeMemCacheLimit       int
	SmallMemCacheLimit       int
	LargeMemCacheObjectLimit int
	SmallMemCacheObjectLimit int

	DebugMode bool
}

type simpleBlogHandler func(*GlobalData, http.ResponseWriter, *http.Request, map[string]string)

func handlerWrapper(handler simpleBlogHandler, globalData *GlobalData) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		logger.Printf("%s %s", r.Method, r.RequestURI)
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
		urlParams["file"] = filepath.Join(prefix, httppath.Clean(urlParams["file"]))
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

func LogFlusher(w *bufio.Writer, period time.Duration, quit chan int) {
	ticker := time.NewTicker(period)
	for {
		select {
		case <-ticker.C:
			w.Flush()
		case <-quit:
			ticker.Stop()
			break
		}
	}
}

func setup() (router *httptreemux.TreeMux, cleanup func()) {
	config = &Config{
		LogFlushPeriod: 2,
		Port:           80,
		// Large memory cache uses 64 MiB at most, with the largest object being 8 MiB.
		LargeMemCacheLimit:       64 * 1024 * 1024,
		LargeMemCacheObjectLimit: 8 * 1024 * 1024,
		// Small memory cache uses 16 MiB at most, with the largest object being 16KiB.
		SmallMemCacheLimit:       16 * 1024 * 1024,
		SmallMemCacheObjectLimit: 16 * 1024,
	}
	confFile := os.Getenv("SIMPLEBLOG_CONFFILE")
	if len(os.Args) > 1 {
		confFile = os.Args[1]
	}

	if confFile == "" {
		confFile = "simpleblog.conf"
	}

	var confReader io.Reader = os.Stdin
	var err error
	if confFile != "-" {
		// Load from stdin
		confReader, err = os.Open(confFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
			os.Exit(1)
		}
	}

	err = goconfig.Load(config, confReader, "SIMPLEBLOG")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	debugMode = config.DebugMode

	logFile, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open log file %s\n", config.LogFile)
		os.Exit(1)
	}

	logBuffer := bufio.NewWriter(logFile)

	logFlusherQuit := make(chan int)
	go LogFlusher(logBuffer,
		time.Duration(config.LogFlushPeriod)*time.Second,
		logFlusherQuit)

	closer := func() {
		logFlusherQuit <- 1
		logger.Println("Shutting down...")
		logBuffer.Flush()
		logFile.Sync()
		logFile.Close()
	}

	var logWriter io.Writer = logBuffer
	if debugMode {
		// In debug mode, use unbuffered logging so that they come out right away.
		logWriter = logFile
	}

	logger = log.New(logWriter, config.LogPrefix, log.LstdFlags)
	debugLogger = log.New(logWriter, "DEBUG ", log.LstdFlags)
	logger.Printf("Starting with config\n%+v\n", config)

	if config.Port != 80 {
		config.Domain = fmt.Sprintf("%s:%d", config.Domain, config.Port)
	}

	diskCache, err := gocache.NewDiskCache(config.CacheDir)
	if err != nil {
		logger.Fatal("Could not create disk cache in", config.CacheDir)
	}

	if !isDirectory(config.DataDir) {
		logger.Fatal("Could not find data directory", config.DataDir)
	}

	if !isDirectory(config.PostsDir) {
		logger.Fatal("Could not find posts directory", config.PostsDir)
	}

	if !isDirectory(filepath.Join(config.DataDir, "assets")) {
		logger.Fatal("Could not find assets directory", filepath.Join(config.DataDir, "assets"))
	}

	if !isDirectory(filepath.Join(config.DataDir, "images")) {
		logger.Fatal("Could not find assets directory", filepath.Join(config.DataDir, "images"))
	}

	largeObjectLimit := config.LargeMemCacheObjectLimit
	largeMemCache := gocache.NewMemoryCache(
		config.LargeMemCacheLimit, largeObjectLimit)

	smallObjectLimit := config.SmallMemCacheObjectLimit
	smallMemCache := gocache.NewMemoryCache(
		config.SmallMemCacheLimit, smallObjectLimit)

	// Create a split cache, putting all objects smaller than 16 KiB into the small cache.
	// This split cache prevents a few large objects from evicting all the smaller objects.
	memCache := gocache.NewSplitSize(
		gocache.SplitSizeChild{MaxSize: smallObjectLimit, Cache: smallMemCache},
		gocache.SplitSizeChild{MaxSize: largeObjectLimit, Cache: largeMemCache})

	multiLevelCache := gocache.MultiLevel{0: memCache, 1: diskCache}

	templates, err := createTemplates()
	if err != nil {
		logger.Fatal("Error parsing template:", err.Error())
	}

	os.Remove(config.TagsPath)
	globalData := &GlobalData{
		RWMutex:   &sync.RWMutex{},
		cache:     multiLevelCache,
		memCache:  memCache,
		templates: templates,
	}

	archive, err := NewArchiveSpecList(config.PostsDir)
	if err != nil {
		logger.Fatal("Could not create archive list: ", err)
	}
	globalData.archive = archive

	go watchFiles(globalData)

	router = httptreemux.New()
	router.PanicHandler = httptreemux.ShowErrorsPanicHandler

	router.GET("/", handlerWrapper(indexHandler, globalData))
	router.GET("/:year/:month/", handlerWrapper(archiveHandler, globalData))
	router.GET("/:year/:month/:post", handlerWrapper(postHandler, globalData))

	router.GET("/images/*file", filePrefixWrapper("images",
		handlerWrapper(staticNoCompressHandler, globalData)))
	router.GET("/assets/*file", filePrefixWrapper("assets",
		handlerWrapper(staticCompressHandler, globalData)))

	router.GET("/tag/:tag", handlerWrapper(tagHandler, globalData))
	// No pagination yet.
	//router.GET("/tag/:tag/:page", handlerWrapper(tagHandler, globalData))

	router.GET("/:page", handlerWrapper(pageHandler, globalData))
	router.GET("/favicon.ico", fileWrapper("assets/favicon.ico",
		handlerWrapper(staticCompressHandler, globalData)))
	router.GET("/feed", handlerWrapper(atomHandler, globalData))

	return router, closer
}

func main() {
	router, closer := setup()

	catchSIGINT(closer, true)
	defer closer()

	logger.Println(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), router))
}
