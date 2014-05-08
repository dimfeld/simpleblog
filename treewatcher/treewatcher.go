package treewatcher

import (
	"github.com/howeyc/fsnotify"
	"os"
	"path/filepath"
)

// TreeWatcher is a wrapper around fsnotify.Watcher that monitors all directories
// within a tree, and automatically adds watches on newly created directories under
// the tree. All events are passed to the caller.
type TreeWatcher struct {
	Event chan *fsnotify.FileEvent
	Error chan error

	quit    chan int
	watcher *fsnotify.Watcher
}

func (tw *TreeWatcher) fsNotifyHandler() {
	for {
		select {
		case event := <-tw.watcher.Event:
			tw.Event <- event

			if event.IsCreate() {
				stat, err := os.Stat(event.Name)
				if err != nil {
					continue
				}

				if stat.IsDir() {
					tw.WatchTree(event.Name)
				}
			}
		case err := <-tw.watcher.Error:
			tw.Error <- err

		case <-tw.quit:
			return
		}
	}
}

func (tw *TreeWatcher) Close() {
	tw.watcher.Close()
	tw.quit <- 1
}

func (tw *TreeWatcher) WatchTree(path string) {
	tw.watcher.Watch(path)
	filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if info.IsDir() {
			tw.watcher.Watch(file)
		}
		return nil
	})
}

func (tw *TreeWatcher) Watch(path string) error {
	return tw.watcher.Watch(path)
}

func New() (*TreeWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	eventChan := make(chan *fsnotify.FileEvent, 10)
	errorChan := make(chan error, 10)
	quitChan := make(chan int)

	tw := &TreeWatcher{eventChan, errorChan, quitChan, watcher}
	go tw.fsNotifyHandler()
	return tw, nil
}
