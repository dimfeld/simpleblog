package main

import (
	"bytes"
	"github.com/dimfeld/gocache"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

func error404(w http.ResponseWriter, r *http.Request) {
	// TODO Real error page here.
	http.NotFound(w, r)
}

func error500(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	if os.IsNotExist(err) {
		error404(w, r)
	} else {
		logger.Println(err)
		error500(w, r)
	}
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

	data, err := globalData.cache.Get(filePath,
		PageSpec{globalData, false, generatePostPage, urlParams})
	if err != nil {
		handleError(w, r, err)
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

	data, err := globalData.cache.Get(filePath,
		PageSpec{globalData, false, generateArchivePage, urlParams})
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

	data, err := globalData.cache.Get(filePath,
		PageSpec{globalData, false, generateTagsPage, urlParams})
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

	data, err := globalData.cache.Get(filePath,
		PageSpec{globalData, false, generateIndexPage, urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, filename, data)
}

func pageHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	page := urlParams["page"]

	filePath := determineCompression(w, r, page)
	object, err := globalData.cache.Get(filePath,
		PageSpec{globalData, true, generateCustomPage, urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, urlParams["page"], object)
}

func staticCompressHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {
	p := path.Clean(urlParams["file"])
	filePath := path.Join("assets", p)
	filePath = determineCompression(w, r, filePath)

	object, err := globalData.cache.Get(filePath,
		DirectCacheFiller{globalData, true})
	if err != nil {
		handleError(w, r, err)
		return
	}

	setStaticAssetHeaders(w)
	sendData(w, r, urlParams["file"], object)
}

func staticNoCompressHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {
	filePath := "content" + path.Clean(r.URL.Path)

	// Only read from the memCache, not the disk cache, since we aren't generating
	// compressed versions.
	object, err := globalData.memCache.Get(filePath,
		DirectCacheFiller{globalData, false})
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
func sendData(w http.ResponseWriter, r *http.Request, name string, object gocache.Object) {
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
	globalData  *GlobalData
	canCompress bool
}

func (d DirectCacheFiller) Fill(cacheObj gocache.Cache, pathStr string) (gocache.Object, error) {
	compressed := false
	if d.canCompress && strings.HasSuffix(pathStr, ".gz") {
		// Get the path without .gz at the end since we start with the uncompresed version.
		pathStr = pathStr[0 : len(pathStr)-3]
		compressed = true
	}

	f, err := d.globalData.dataDir.Open(pathStr)
	if err != nil {
		return gocache.Object{}, err
	}
	defer f.Close()

	fstat, err := f.Stat()
	if err != nil {
		return gocache.Object{}, err
	}

	data := make([]byte, fstat.Size())
	_, err = f.Read(data)
	if err != nil {
		return gocache.Object{}, err
	}

	if d.canCompress {
		compressedObj, uncompressedObj, err := gocache.CompressAndSet(cacheObj, pathStr, data, fstat.ModTime())
		if compressed {
			return compressedObj, err
		} else {
			return uncompressedObj, err
		}
	} else {
		obj := gocache.Object{Data: data, ModTime: fstat.ModTime()}
		cacheObj.Set(pathStr, obj)
		return obj, nil
	}
}
