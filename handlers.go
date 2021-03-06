package main

import (
	"bytes"
	"github.com/dimfeld/glog"
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
		glog.Warningf("%s from %s, referrer %s: err %s",
			r.URL.Path, r.RemoteAddr, r.Referer(), err)
		error404(w, r)
	} else {
		glog.Errorln(r.URL.Path, ":", err)
		error500(w, r)
	}
}

// determineCompression figures out if compression can be used, and adds a .gz extension so that
// we get the compressed version of the file instead.
func determineCompression(w http.ResponseWriter, r *http.Request, path string) (outPath string,
	compression bool) {

	if _, ok := r.Header["Range"]; ok {
		// No compression if the user passed a range request, since returning a slice of the
		// compressed version from the cache would then return invalid data.
		return path, false
	}

	encodings := r.Header["Accept-Encoding"]
	outPath = path
	for index := range encodings {
		if strings.Contains(encodings[index], "gzip") {
			// Use the gzipped version.
			return path + ".gz", true
		}
	}

	return path, false
}

func postHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	filePath := path.Join(urlParams["year"], urlParams["month"], urlParams["post"]) + ".md"
	filePath, compression := determineCompression(w, r, filePath)

	data, err := globalData.cache.Get(filePath,
		PageSpec{globalData: globalData, customPage: false,
			generator: generatePostPage, params: urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, urlParams["post"]+".html", compression, data)
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
	filePath, compression := determineCompression(w, r, filePath)

	data, err := globalData.cache.Get(filePath,
		PageSpec{globalData: globalData, customPage: false,
			generator: generateArchivePage, params: urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, filename+".html", compression, data)
}

func tagHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	filePath := path.Join("tags", urlParams["tag"])
	filePath, compression := determineCompression(w, r, filePath)

	data, err := globalData.cache.Get(filePath,
		PageSpec{globalData: globalData, customPage: false,
			generator: generateTagsPage, params: urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, urlParams["tag"]+".html", compression, data)
}

func indexHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	filename := "index.html"
	filePath, compression := determineCompression(w, r, filename)

	data, err := globalData.cache.Get(filePath,
		PageSpec{globalData: globalData, customPage: false,
			generator: generateIndexPage, params: urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, filename, compression, data)
}

func pageHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	page := urlParams["page"]

	filePath, compression := determineCompression(w, r, page)
	object, err := globalData.cache.Get(filePath,
		PageSpec{globalData: globalData, customPage: true,
			generator: generateCustomPage, params: urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, urlParams["page"]+".html", compression, object)
}

func atomHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {

	filename := "atom.xml"
	filePath, compression := determineCompression(w, r, filename)

	object, err := globalData.cache.Get(filePath,
		PageSpec{globalData: globalData, customTemplate: "atom.tmpl.html",
			generator: generateIndexPage, params: urlParams})
	if err != nil {
		handleError(w, r, err)
		return
	}

	sendData(w, r, filename, compression, object)
}

func staticCompressHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {
	filePath := urlParams["file"]
	filePath, compression := determineCompression(w, r, filePath)

	if glog.V(1) {
		glog.Infoln("Getting path", filePath)
	}
	object, err := globalData.cache.Get(filePath,
		DirectCacheFiller{globalData, true})
	if err != nil {
		handleError(w, r, err)
		return
	}

	setStaticAssetHeaders(w)
	sendData(w, r, urlParams["file"], compression, object)
}

func staticNoCompressHandler(globalData *GlobalData, w http.ResponseWriter,
	r *http.Request, urlParams map[string]string) {
	filePath := urlParams["file"]

	// Only read from the memCache, not the disk cache, since we aren't generating
	// compressed versions.
	object, err := globalData.memCache.Get(filePath,
		DirectCacheFiller{globalData, false})
	if err != nil {
		handleError(w, r, err)
		return
	}

	setStaticAssetHeaders(w)
	sendData(w, r, filePath, false, object)
}

func setStaticAssetHeaders(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Now().AddDate(0, 0, 1).String())
	// One day in seconds
	w.Header().Set("Cache-Control", "public, max-age=86400")
}

// sendData returns a file to the user, handling relevant headers in the request and response.
func sendData(w http.ResponseWriter, r *http.Request, name string,
	compression bool, object gocache.Object) {

	header := w.Header()
	header.Add("Vary", "Accept-Encoding")
	// 5 minutes in seconds
	if _, ok := header["Cache-Control"]; !ok {
		header.Set("Cache-Control", "public, max-age=300")
	}
	if _, ok := header["Expires"]; !ok {
		header.Set("Expires", time.Now().Add(300*time.Second).String())
	}

	if compression {
		w.Header().Set("Content-Encoding", "gzip")
	}

	if glog.V(1) {
		glog.Infof("Sending data for %s%s [%d]\n",
			name,
			func() string {
				if compression {
					return ".gz"
				} else {
					return ""
				}
			}(),
			len(object.Data),
		)
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

	f, err := http.Dir(config.DataDir).Open(pathStr)
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
		uncompressedObj, compressedObj, err := gocache.CompressAndSet(cacheObj, pathStr, data, fstat.ModTime())
		if glog.V(2) {
			glog.Infof("%s: compressed %d, uncompressed %d",
				pathStr, len(compressedObj.Data), len(uncompressedObj.Data))
		}
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
