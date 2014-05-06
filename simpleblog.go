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
	memCache := cachePkg.NewMemoryCache(64*1024*1024, diskCache)

	globalData := &GlobalData{cache: memCache}

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
	router.GET("/favicon.ico", handlerWrapper(simpleHandler, globalData))

	http.ListenAndServe(":8080", router)
}
