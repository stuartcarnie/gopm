// TODO when the code in os_darwin.go can pass tests,
// this file should be excluded for darwin.

package fswatcher

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type FSWatcher struct {
	eventCh chan Event
	errCh   chan error
	mw      *fsnotify.Watcher
	closed  chan struct{}
	done    chan struct{}

	// activeWatchers is a map keyed by directory paths for directories that has been
	// added to watch. The bool value is used to dedup multiple remove events for the same
	// directory from fsnotify (inotify sends two events when removing a dir, "DELETE_SELF"
	// and "DELETE,ISDIR" where the difference isn't exposed by fsnotify). The value is true
	// if the watcher is active, and false for directories that no longer are watched.
	// By only sending events on the transition from true to false we ensure that only one
	// event is sent even when we get duplicate remove events. The key must be removed if
	// a file is created with the same name as a previous directory.
	activeWatches map[string]bool
}

// populateWatches is used to walk the provided path and add/remove directories to the
// underlying fsnotify watcher since it isn't recursive. It return all files found in
// watched directories during the walk. If the initPath is a newly created directory
// we must also send create events for the returned files to prevent a race condition where
// files are created before the directory is watched.
func (w *FSWatcher) populateWatches(initPath string, selectPath func(string, os.FileInfo) bool) ([]string, error) {
	var files []string
	err := filepath.Walk(initPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// fast path for directories that are already watched
		if w.activeWatches[path] {
			return nil
		}

		if info.IsDir() && !selectPath(path, info) {
			if v, ok := w.activeWatches[path]; ok && v {
				w.mw.Remove(path)
				w.activeWatches[path] = false
			}
			return filepath.SkipDir
		}

		if !info.IsDir() {
			files = append(files, path)
			return nil
		}

		if v, ok := w.activeWatches[path]; !ok || !v {
			if err := w.mw.Add(path); err != nil {
				return err
			}
			w.activeWatches[path] = true
			// When mw.Add returns the folder might already contain files and/or
			// directories that we must send events for manually. The recommended
			// way in inotify(7) is to:
			//
			// "[...] new files (and subdirectories) may already exist inside the
			// subdirectory. Therefore, you might want to scan the contents of ths
			// subdirectory immediately after adding the watch (and, if desired,
			// recursively add watchers for any subdirectories that it contains).
			fs, err := w.populateWatches(path, selectPath)
			files = append(files, fs...)
			if err != nil {
				return err
			}
			// We must stop walking here since we initiated a new walk above, to avoid
			// duplicate events..
			return filepath.SkipDir
		}
		return nil
	})
	return files, err
}

func newFSWatcher(root string, selectPath func(path string, info os.FileInfo) bool) (*FSWatcher, error) {
	mw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create new watcher: %v", err)
	}

	eventCh := make(chan Event)
	w := &FSWatcher{
		eventCh:       eventCh,
		mw:            mw,
		closed:        make(chan struct{}),
		done:          make(chan struct{}),
		activeWatches: make(map[string]bool),
	}

	if _, err := w.populateWatches(root, selectPath); err != nil {
		return nil, fmt.Errorf("initial root walk failed: %w", err)
	}

	go func() {
		defer close(eventCh)
		defer close(w.done)
		for e := range mw.Events {
			path := e.Name
			switch {
			// fsnotify processes file renaming as a Rename event followed by a
			// Create event, so we can effectively treat renaming as removal.
			case e.Op&(fsnotify.Rename|fsnotify.Remove) != 0:
				if _, ok := w.activeWatches[path]; ok {
					// We try to avoid sending remove events for directories, however
					// fsnotify sends two events per delete. One for the "DELETE_SELF",
					// and another for "DELETE,ISDIR". These flags aren't exposed by
					// fsnotify so after the file/directory has been removed there is
					// no way to tell if the path was a directory.
					// To prevent events for the second we just set the entry to false
					// w/o dropping it entirely.
					w.activeWatches[path] = false
					continue
				}
				w.send(Event{path, Removed})
			case e.Op&(fsnotify.Chmod|fsnotify.Write|fsnotify.Create) != 0:
				var op Op
				if (e.Op & fsnotify.Create) != 0 {
					op = Created
				} else {
					op = Changed
				}

				di, err := os.Stat(path)
				if err != nil {
					// This might happen when a file has been created and removed
					// immediately afterwards.
					continue
				}
				if !di.IsDir() {
					// We know for sure that this isn't a directory so we must remove
					// any entry from activeWatchers in case we have had a directory
					// with the same name earlier.
					delete(w.activeWatches, path)
					if selectPath(path, di) {
						w.send(Event{path, op})
					}
					continue
				}
				files, err := w.populateWatches(path, selectPath)
				if err != nil {
					continue
				}
				// If the directory was just created, we need to send file creation events
				// for all files created as well.
				if op == Created && w.activeWatches[path] {
					for _, f := range files {
						w.send(Event{f, op})
					}
				}
			}
		}
	}()

	return w, nil
}

func (w *FSWatcher) send(e Event) {
	select {
	case w.eventCh <- e:
		return
	case <-w.closed:
	}
}

// Close closes the watcher. Nothing more will be sent
// on the Events channel after this returns.
func (w *FSWatcher) Close() error {
	w.mw.Close()
	close(w.closed)
	<-w.done
	return nil
}

// Events returns a channel on which update events can be received.
func (w *FSWatcher) Events() <-chan Event {
	return w.eventCh
}
