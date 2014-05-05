package main

import (
	"github.com/dimfeld/simpleblog/cache"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func error404(w http.ResponseWriter, r *http.Request, path string) {
	// TODO Real error page here.
	http.NotFound(w, r)
	return
}

func determineCompression(w http.ResponseWriter, r *http.Request, path string) (outPath string) {
	encodings := r.Header.Get("Accept-Encoding")
	outPath = path
	if strings.Contains(encodings, "gzip") {
		outPath = path + ".gz"
		w.Header().Set("Content-Encoding", "gzip")
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

	sendData(w, r, data)
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

	w.Header().Set("Expires", time.Now().AddDate(1, 0, 0).String())
	// One year in seconds
	w.Header().Set("Cache-Control", "public, max-age=31536000")

	object, err := globalData.cache.Get(filePath, DirectCacheFiller{})
	if err != nil {
		// TODO 404 error
	}

	sendData(w, r, object)
}

// sendData returns a file to the user, handling relevant headers in the request and response.
func sendData(w http.ResponseWriter, r *http.Request, object cache.Object) error {
	header := w.Header()
	header.Add("Vary", "Accept-Encoding")
	modTime := object.ModTime
	// 30 days in seconds
	if _, ok := header["Cache-Control"]; !ok {
		header.Add("Cache-Control", "public, max-age=2592000")
	}
	if _, ok := header["Expires"]; !ok {
		header.Set("Expires", time.Now().AddDate(0, 1, 0).String())
	}

	writeData := true
	if !modTime.IsZero() {
		header.Set("Last-Modified", modTime.String())

		if modifiedSinceStr := r.Header.Get("If-Modified-Since"); modifiedSinceStr != "" {
			sinceTime, err := http.ParseTime(modifiedSinceStr)
			if err == nil && sinceTime.After(modTime) {
				writeData = false
			}
		}
	}

	if writeData {
		header.Set("Content-Length", strconv.Itoa(len(object.Data)))
		w.Write(object.Data)
	} else {
		w.WriteHeader(http.StatusNotModified)
	}

	return nil
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
