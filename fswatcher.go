package main

import (
	"github.com/howeyc/fsnotify"
	"os"
	"path/filepath"
)

// initFsWatcher sets up the paths to watch.
func initFsWatcher() Error {

}

// handleChange invalidates relevant parts of the cache when something changes on the file system.
func handleChange(path string) {

}

func purgeVarnish(path string) error {

}

func purgeCache(path string) error {
	path = "cache/" + path
	matches, err := filepath.Glob(path)

	if err != nil {
		return
	}

	for file := range matches {
		err = os.RemoveAll(file)
		if err {
			// Syslog removal failure
		}
	}
}
