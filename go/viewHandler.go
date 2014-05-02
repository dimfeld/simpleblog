package main

import (
	"net/http"
	"os"
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

func viewHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]

	secure := secureFilename(path)
	if !secure {
		error404(w, r, path)
		return
	}

	filename := mapFilename(path)

	err = sendFile(filename)
	if err != nil {
		error404(w, r, path)
		return
	}
}

func secureFilename(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}

	// What else here? There must be some standard library for this.

	return true
}

// mapFilename takes the user-supplied path and figures out what to actually look up.
func mapFilename(path string) string {
	// Ugh, this whole thing is a harcoded mess. But it's simple...

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
	cachedPath = findCachedFile(filename)
	if cachedPath != nil {
		http.ServeFile(w, r, cachedPath)
		return
	}

	// If it's not in the cache, see if it can be generated
	cachedPath, err = generateCachedFile(filename)
	if err != nil {
		return err
	}

	http.ServeFile(w, r, cachedPath)
}

// findCachedPath returns the full path for passing into ServeFile, if the file is cached.
func findCachedPath(path string) string {
	cachePath = "cache/" + path
	if exists(cachePath) {
		return cachePath
	}
	return nil
}

// generateCachedFile creates HTML output for the user's request and places the file into the cache.
func generateCachedFile(filename string) (string, Error) {
	return nil, ""
}
