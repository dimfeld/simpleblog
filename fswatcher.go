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

	logger.Println("Watching directory", config.DataDir)
	tw.WatchTree(config.DataDir)

	if _, err = filepath.Rel(config.DataDir, config.PostsDir); err != nil {
		// PostsDir is not a subdirectory of dataDir, so watch it too.
		logger.Println("Watching directory", string(config.PostsDir))
		tw.WatchTree(string(config.PostsDir))
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
	cachePath, err := filepath.Rel(config.PostsDir, fullPath)
	isPost := true
	if err != nil || strings.HasPrefix(cachePath, "..") {
		cachePath, err = filepath.Rel(config.DataDir, fullPath)
		isPost = false
		if err != nil || strings.HasPrefix(cachePath, "..") {
			logger.Printf("Path %s is not in data or posts dir")
			return
		}
	}

	// We just keep it simple and clear the entire cache when a post is updated, since it's likely
	// that we need to update the tag list on every page, or something similar.
	// Same for when a template is updated since that affects every page.
	templateUpdate := strings.Contains(cachePath, "templates/")
	if isPost || templateUpdate {
		debug("FsWatcher clearing post data for update of", cachePath)

		newArchiveList, err := NewArchiveSpecList(config.PostsDir)
		if err != nil {
			newArchiveList = nil
		}

		templates, err := createTemplates()
		if err != nil {
			logger.Println("Error parsing template:", err.Error())
		}

		globalData.Lock()
		globalData.archive = newArchiveList
		if templateUpdate {
			globalData.templates = templates
		}
		globalData.Unlock()

		os.Remove(config.TagsPath)

		globalData.cache.Del("*")

	} else {
		// It's some other data, so just invalidate that one object from the cache.
		debug("FsWatcher clearing data for", cachePath)
		globalData.cache.Del(cachePath)
		globalData.cache.Del(cachePath + ".gz")
	}
}
