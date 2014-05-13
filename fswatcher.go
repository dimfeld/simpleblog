package main

import (
	"github.com/dimfeld/simpleblog/treewatcher"
	"github.com/howeyc/fsnotify"
	"os"
	"path/filepath"
	"strings"
)

func watchFiles(globalData *GlobalData) {
	tw, err := treewatcher.New()
	if err != nil {
		return
	}

	tw.WatchTree(string(globalData.dataDir))
	tw.WatchTree(string(globalData.postsDir))

	for {
		select {
		case event := <-tw.Event:
			handleFileEvent(globalData, event)
		case err := <-tw.Error:
			logger.Println("Fswatcher error:", err)
		}
	}
}

func handleFileEvent(globalData *GlobalData, event *fsnotify.FileEvent) {
	fullPath := event.Name

	// Get the cache path, relative to either the data directory or the post directory.
	cachePath, err := filepath.Rel(string(globalData.dataDir), fullPath)
	isPost := false
	if err != nil {
		cachePath, err = filepath.Rel(globalData.postsDir, fullPath)
		isPost = true
		if err != nil {
			logger.Printf("Path %s is not in data or posts dir")
			return
		}
	}

	// We just keep it simple and clear the entire cache when a post is updated, since it's likely
	// that we need to update the tag list on every page, or something similar.
	// Same for when a template is updated since that affects every page.
	if isPost || strings.HasSuffix(fullPath, "tmpl.html") {
		globalData.cache.Del("*")
		os.Remove(globalData.tagsPath)
	} else {
		// It's some other data, so just invalidate that one object from the cache.
		globalData.cache.Del(cachePath)
		globalData.cache.Del(cachePath + ".gz")
	}
}
