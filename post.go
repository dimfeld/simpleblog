package main

import (
	"os"
)

type Post struct {
	sourcePath string
	timestamp  time.Time
	title      string
	tags       []string
	content    string
}

func NewPost(filePath string, readContent bool) (*Post, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

}
