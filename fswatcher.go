package main

import (
	"github.com/dimfeld/treewatcher"
	"github.com/howeyc/fsnotify"
	"os"
	"path/filepath"
	"strings"
)

func watchFiles(globalData *GlobalData) {
	tw, err := treewatcher.New()
	if err != nil {
		logger.Fatal("Failed to create file system watcher")
	}

	logger.Println("Watching directory", string(globalData.dataDir))
	tw.WatchTree(string(globalData.dataDir))

	if _, err = filepath.Rel(string(globalData.dataDir), globalData.postsDir); err != nil {
		// PostsDir is not a subdirectory of dataDir, so watch it too.
		logger.Println("Watching directory", string(globalData.postsDir))
		tw.WatchTree(string(globalData.postsDir))
	}

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
	cachePath, err := filepath.Rel(globalData.postsDir, fullPath)
	isPost := true
	if err != nil {
		cachePath, err = filepath.Rel(string(globalData.dataDir), fullPath)
		isPost = false
		if err != nil {
			logger.Printf("Path %s is not in data or posts dir")
			return
		}
	}

	// We just keep it simple and clear the entire cache when a post is updated, since it's likely
	// that we need to update the tag list on every page, or something similar.
	// Same for when a template is updated since that affects every page.
	if isPost || strings.HasSuffix(fullPath, "tmpl.html") {
		debug("FsWatcher clearing post data for update of", fullPath)

		newArchiveList, err := NewArchiveSpecList(globalData.postsDir)
		if err != nil {
			newArchiveList = nil
		}

		globalData.Lock()
		globalData.archive = newArchiveList
		globalData.Unlock()

		os.Remove(globalData.tagsPath)

		globalData.cache.Del("*")

	} else {
		// It's some other data, so just invalidate that one object from the cache.
		debug("FsWatcher clearing data for", fullPath)
		globalData.cache.Del(cachePath)
		globalData.cache.Del(cachePath + ".gz")
	}
}
