package main

import (
	"flag"
	"fmt"
	"github.com/dimfeld/glog"
	"github.com/dimfeld/gocache"
	"github.com/dimfeld/goconfig"
	"github.com/dimfeld/httppath"
	"github.com/dimfeld/httptreemux"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"
)

var (
	config *Config
)

func catchSIGINT(f func(), quit bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			glog.Infoln("SIGINT received...")
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

	LogDir string

	Domain string
	Port   int

	LargeMemCacheLimit       int
	SmallMemCacheLimit       int
	LargeMemCacheObjectLimit int
	SmallMemCacheObjectLimit int
}

type simpleBlogHandler func(*GlobalData, http.ResponseWriter, *http.Request, map[string]string)

func handlerWrapper(handler simpleBlogHandler, globalData *GlobalData) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		glog.Infof("%s %s", r.Method, r.RequestURI)
		startTime := time.Now()
		handler(globalData, w, r, urlParams)
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		glog.Infof("   Handled in %d us", duration/time.Microsecond)
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

func setup() (router *httptreemux.TreeMux, cleanup func()) {
	flag.Parse()
	config = &Config{
		Port: 80,
		// Large memory cache uses 64 MiB at most, with the largest object being 8 MiB.
		LargeMemCacheLimit:       64 * 1024 * 1024,
		LargeMemCacheObjectLimit: 8 * 1024 * 1024,
		// Small memory cache uses 16 MiB at most, with the largest object being 16KiB.
		SmallMemCacheLimit:       16 * 1024 * 1024,
		SmallMemCacheObjectLimit: 16 * 1024,
	}
	confFile := os.Getenv("SIMPLEBLOG_CONF")
	if confFile == "" && flag.NArg() != 0 {
		confFile = flag.Arg(0)
	}

	if confFile == "" {
		confFile = os.Args[0] + ".conf"
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

	// Use config.LogDir if not given on the command line.
	dir := flag.CommandLine.Lookup("log_dir")
	if dir != nil && dir.Value.String() == "" {
		if config.LogDir == "" {
			config.LogDir = "."
		}
		flag.Set("log_dir", config.LogDir)
	}

	closer := func() {
		glog.Infoln("Shutting down...")
		glog.Flush()
	}

	glog.Infoln("Starting with config\n%+v\n", config)

	if config.Port != 80 {
		config.Domain = fmt.Sprintf("%s:%d", config.Domain, config.Port)
	}

	diskCache, err := gocache.NewDiskCache(config.CacheDir)
	if err != nil {
		glog.Fatal("Could not create disk cache in", config.CacheDir)
	}

	if !isDirectory(config.DataDir) {
		glog.Fatal("Could not find data directory", config.DataDir)
	}

	if !isDirectory(config.PostsDir) {
		glog.Fatal("Could not find posts directory", config.PostsDir)
	}

	if !isDirectory(filepath.Join(config.DataDir, "assets")) {
		glog.Fatal("Could not find assets directory", filepath.Join(config.DataDir, "assets"))
	}

	if !isDirectory(filepath.Join(config.DataDir, "images")) {
		glog.Fatal("Could not find assets directory", filepath.Join(config.DataDir, "images"))
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
		glog.Fatal("Error parsing template:", err.Error())
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
		glog.Fatal("Could not create archive list: ", err)
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

	glog.Infoln(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), router))
}
