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

func error404(w http.ResponseWriter, r *http.Request, path string) {
	// TODO Real error page here.
	http.NotFound(w, r)
	return
}

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
}

func tagHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

}

func indexHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func staticCompressHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {
	filePath := path.Join("assets", urlParams["file"])
	filePath = determineCompression(w, r, filePath)

	makeStaticAssetHeaders(w)

	object, err := globalData.cache.Get(filePath, DirectCacheFiller{})
	if err != nil {
		// TODO 404 error
	}

	sendData(w, r, urlParams["file"], object)
}

func simpleHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {
	path := "content" + r.URL.Path
	makeStaticAssetHeaders(w)
	http.ServeFile(w, r, path)
}

func makeStaticAssetHeaders(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Now().AddDate(1, 0, 0).String())
	// One year in seconds
	w.Header().Set("Cache-Control", "public, max-age=31536000")
}

// sendData returns a file to the user, handling relevant headers in the request and response.
func sendData(w http.ResponseWriter, r *http.Request, name string, object cache.Object) {
	header := w.Header()
	header.Add("Vary", "Accept-Encoding")
	modTime := object.ModTime
	// 30 days in seconds
	if _, ok := header["Cache-Control"]; !ok {
		header.Set("Cache-Control", "public, max-age=2592000")
	}
	if _, ok := header["Expires"]; !ok {
		header.Set("Expires", time.Now().AddDate(0, 1, 0).String())
	}

	reader := bytes.NewReader(object.Data)
	http.ServeContent(w, r, name, modTime, reader)
}

type DirectCacheFiller struct {
}

func (d DirectCacheFiller) Fill(cacheObj cache.Cache, pathStr string) (cache.Object, error) {
	compressed := false
	if strings.HasSuffix(pathStr, ".gz") {
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

	compressedObj, uncompressedObj, err := cache.CompressAndSet(cacheObj, pathStr, data, fstat.ModTime())
	if compressed {
		return compressedObj, err
	} else {
		return uncompressedObj, err
	}
}
