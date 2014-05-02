package main

import (
	"compress/gzip"
	"net/http"
	"os"
	"path"
	"strings"
)

func error404(w http.ResponseWriter, r *http.Request, path string) {
	w.WriteHeader(http.StatusNotFound)
	err = sendFile(w, r, "errorPages/404.html")
	if err != nil {
		// Ouch, couldn't find our 404 page.
		http.NotFound(w, r)
	}
	return
}

func canCompress(r *http.Request) bool {
	encodings := r.Header.Get("Accept-Encoding")
	return strings.Contains(encodings, "gzip")
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]

	filename, useCache := mapFilename(path)

	err = sendFile(filename)
	if err != nil {
		error404(w, r, path)
		return
	}
}

// mapFilename takes the user-supplied path and figures out what to actually look up.
func mapFilename(path string) (filename string, useCache bool) {
	// Ugh, this whole thing is a harcoded mess. But it's simple...

	useCache = true

	// Root path
	if len(path) == 0 {
		return "index"
	}

	// TODO Does this look like a non-post page?
	pagePath := "pages/" + path
	if exists(pagePath) {
		return pagePath
	}

	// TODO Does this look like a post page?
	postPath := "posts/" + path
	if exists(postPath) {
		return postPath
	}

	// TODO Does this look like an archive page?

	pathLen := len(path)
	if pathLen == 7 || pathLen == 8 {
		archivePath = "archives/" + path[0:5] + "-" + path[6:8]
		if exists(archivePath) {
			return archivePath
		}
	}

	// Nothing special to do with this path.
	return path
}

func exists(path string) bool {
	_, err = os.Stat(path)
	return !os.IsNotExist(err)
}

// sendFile returns a file to the user, generating it into the cache if needed.
func sendFile(w http.ResponseWriter, r *http.Request, filename string) Error {
	// Check the cache.
	cachedPath = findCachedPath(r, filename)
	if cachedPath != nil {
		http.ServeFile(w, r, cachedPath)
		return
	}

	// If it's not in the cache, see if it can be generated
	cachedPath, err = generateCachedFile(r, filename)
	if err != nil {
		return err
	}

	http.ServeFile(w, r, cachedPath)
}

// findCachedPath returns the full path for passing into ServeFile, if the file is cached.
func findCachedPath(r *http.Request, path string) string {
	cachePath = "cache/" + path
	if canCompress(r) {
		gzippedCache = cachePath + ".gz"
		if exists(gzippedCache) {
			return gzippedCache
		}
	}
	if exists(cachePath) {
		return cachePath
	}
	return nil
}

// generateCachedFile creates HTML output for the user's request and places the file into the cache.
func generateCachedFile(r *http.Request, path string) (string, Error) {
	path = "cache/" + path
	dirPath := path.Split(path)
	err := os.MkdirAll(dirPath, 0700)
	if err != nil {
		return nil, err
	}

	// Do the templating
	//data := runTemplate(some parameters here)
	data = ""

	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	file.Write(data)

	err = generateGzipCache(path+".gz", data)
	return path, err
}

func generateGzipCache(path string, byte []data) Error {
	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return err
	}

	gz := gzip.NewWriterLevel(file, gzip.BestCompression)
	_, err = gz.Write(data)
	return err
}
