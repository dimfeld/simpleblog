package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
)

type Article struct {
	Title   string
	Tags    []string
	Content []byte
}

func noCacheHandler(w http.ResponseWriter, r *http.Request) {
	path = r.URL.Path[1:]
	if !exists(path) {
		http.Error(w, "File not found", http.StatusNotFound)
	}
	http.ServeFile(w, r, path)
}

func main() {
	// TODO Load configuration

	http.HandleFunc("/", viewHandler)
	http.HandleFunc("/images/", noCacheHandler)
	http.ListenAndServe(":8080", nil)
}
