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

func imageHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	path := r.URL.Path[1:]

	if !exists(path) {
		http.Error(w, "File not found", http.StatusNotFound)
	}
	http.ServeFile(w, r, path)
}

type simpleBlogHandler func(*GlobalData, http.ResponseWriter, *http.Request, map[string]string)

func handlerWrapper(handler simpleBlogHandler, globalData *GlobalData) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, urlParams map[string]string) {
		handler(globalData, w, r, urlParams)
	}
}

func main() {
	// TODO Load configuration

	memCache := cachePkg.NewMemoryCache(64*1024*1024, nil, nil)

	globalData := &GlobalData{cache: memCache}

	router := httprouter.New()
	router.GET("/", handlerWrapper(indexHandler, globalData))
	router.GET("/:year/:month", handlerWrapper(archiveHandler, globalData))
	router.GET("/:year/:month/:post", handlerWrapper(postHandler, globalData))
	router.GET("/tag/:tag", handlerWrapper(tagHandler, globalData))
	// No pagination yet.
	//router.GET("/tag/:tag/:page", handlerWrapper(tagHandler, globalData))
	router.GET("/images/*file", handlerWrapper(imageHandler, globalData))
	router.GET("/assets/*file", handlerWrapper(staticCompressHandler, globalData))

	http.ListenAndServe(":8080", router)
}
