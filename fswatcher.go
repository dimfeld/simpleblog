package main

import (
	"github.com/dimfeld/simpleblog/treewatcher"
	"github.com/howeyc/fsnotify"
	"os"
	"path/filepath"
)

func watchFiles(globalData *GlobalData) {
	tw, err := treewatcher.New()
	if err != nil {
		return
	}

	tw.WatchTree(string(globalData.dataDir))

	for {
		select {
		case event := <-tw.Event:
			handleFileEvent(globalData, event)
		case err := <-tw.Error:
			globalData.logger.Println(err)
		}
	}
}

func handleFileEvent(globalData *GlobalData, event *fsnotify.FileEvent) {
	fullPath := event.Name

	dataPath, err := filepath.Rel(string(globalData.dataDir), fullPath)
	if err != nil {
		globalData.logger.Println(err)
		return
	}

	handleChangeWithoutPost(globalData, fullPath, dataPath)

	if event.IsModify() || event.IsCreate() || event.IsRename() {
		_, err = os.Stat(fullPath)
		if os.IsNotExist(err) {
			return
		}
		handlePost(globalData, fullPath, dataPath)
	}
}

func handleChangeWithoutPost(globalData *GlobalData, fullPath string, dataPath string) {
	globalData.cache.Del(dataPath)
	globalData.cache.Del("index.html")

}

func handlePost(globalData *GlobalData, fullPath string, dataPath string) {
	// Proactively generate the post?
}
