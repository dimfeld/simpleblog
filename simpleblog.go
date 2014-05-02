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

func main() {
	// TODO Load configuration

	http.HandleFunc("/", viewHandler)
	http.ListenAndServe(":8080", nil)
}
