package main

import (
	"bytes"
	"github.com/dimfeld/simpleblog/cache"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

func error404(w http.ResponseWriter, r *http.Request) {
	// TODO Real error page here.
	http.NotFound(w, r)
	return
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	error404(w, r)
}

// determineCompression figures out if compression can be used, and adds a .gz extension so that
// we get the compressed version of the file instead.
func determineCompression(w http.ResponseWriter, r *http.Request, path string) (outPath string) {
	if _, ok := r.Header["Range"]; ok {
		// No compression if the user passed a range request, since returning a slice of the
		// compressed version from the cache would then return invalid data.
		return path
	}

	encodings := r.Header["Accept-Encoding"]
	outPath = path
	for index := range encodings {
		if strings.Contains(encodings[index], "gzip") {
			// Use the gzipped version.
			outPath = path + ".gz"
			w.Header().Set("Content-Encoding", "gzip")
			break
		}
	}

	return outPath
}

func postHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	filePath := path.Join(urlParams["year"], urlParams["month"], urlParams["post"])
	filePath = determineCompression(w, r, filePath)

	data, err := globalData.cache.Get(filePath, PageSpec{generatePostPage, urlParams})
	if err != nil {
		// TODO Handle err
		return
	}

	sendData(w, r, urlParams["post"]+".html", data)
}

func archiveHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	year := urlParams["year"]
	if len(year) == 2 {
		year = "20" + year
	}
	month := urlParams["month"]
	if len(month) == 1 {
		month = "0" + month
	}
	filename := year + "-" + month
	filePath := path.Join("archive", filename)
	filePath = determineCompression(w, r, filePath)

	data, err := globalData.cache.Get(filePath, PageSpec{generateArchivePage, urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, filename+".html", data)
}

func tagHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	filePath := path.Join("tags", urlParams["tag"])
	filePath = determineCompression(w, r, filePath)

	data, err := globalData.cache.Get(filePath, PageSpec{generateTagsPage, urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, urlParams["tag"]+".html", data)
}

func indexHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	filename := "index.html"
	filePath := determineCompression(w, r, filename)

	data, err := globalData.cache.Get(filePath, PageSpec{generateIndexPage, urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, filename, data)
}

func staticCompressHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {
	filePath := path.Join("assets", urlParams["file"])
	filePath = determineCompression(w, r, filePath)

	object, err := globalData.cache.Get(filePath, DirectCacheFiller{true})
	if err != nil {
		handleError(w, r, err)
		return
	}

	setStaticAssetHeaders(w)
	sendData(w, r, urlParams["file"], object)
}

func staticNoCompressHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {
	filePath := "content" + r.URL.Path

	// Only read from the memCache, not the disk cache, since we aren't generating
	// compressed versions.
	object, err := globalData.memCache.Get(filePath, DirectCacheFiller{false})
	if err != nil {
		handleError(w, r, err)
		return
	}

	setStaticAssetHeaders(w)
	sendData(w, r, urlParams["file"], object)
}

func setStaticAssetHeaders(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Now().AddDate(1, 0, 0).String())
	// One year in seconds
	w.Header().Set("Cache-Control", "public, max-age=31536000")
}

// sendData returns a file to the user, handling relevant headers in the request and response.
func sendData(w http.ResponseWriter, r *http.Request, name string, object cache.Object) {
	header := w.Header()
	header.Add("Vary", "Accept-Encoding")
	// 30 days in seconds
	if _, ok := header["Cache-Control"]; !ok {
		header.Set("Cache-Control", "public, max-age=2592000")
	}
	if _, ok := header["Expires"]; !ok {
		header.Set("Expires", time.Now().AddDate(0, 1, 0).String())
	}

	reader := bytes.NewReader(object.Data)
	http.ServeContent(w, r, name, object.ModTime, reader)
}

type DirectCacheFiller struct {
	canCompress bool
}

func (d DirectCacheFiller) Fill(cacheObj cache.Cache, pathStr string) (cache.Object, error) {
	compressed := false
	if d.canCompress && strings.HasSuffix(pathStr, ".gz") {
		// Get the path without .gz at the end since we start with the uncompresed version.
		pathStr = pathStr[0 : len(pathStr)-3]
		compressed = true
	}

	f, err := os.Open(pathStr)
	if err != nil {
		return cache.Object{}, err
	}
	defer f.Close()

	fstat, err := f.Stat()
	if err != nil {
		return cache.Object{}, err
	}

	data := make([]byte, fstat.Size())
	_, err = f.Read(data)
	if err != nil {
		return cache.Object{}, err
	}

	if d.canCompress {
		compressedObj, uncompressedObj, err := cache.CompressAndSet(cacheObj, pathStr, data, fstat.ModTime())
		if compressed {
			return compressedObj, err
		} else {
			return uncompressedObj, err
		}
	} else {
		obj := cache.Object{data, fstat.ModTime()}
		cacheObj.Set(pathStr, obj)
		return obj, nil
	}
}
