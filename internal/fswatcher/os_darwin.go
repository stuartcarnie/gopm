//go:build darwin && ignore
// +build darwin,ignore

// Note: this is disabled for now because the fsevents package doesn't
// seem to work.

package fswatcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/stuartcarnie/gopm/internal/fsevents"
)

const (
	fRemoved = fsevents.ItemRemoved | fsevents.ItemRenamed
	fChanged = fsevents.ItemModified | fsevents.ItemChangeOwner
	fCreated = fsevents.ItemCreated
)

type FSWatcher struct {
	eventCh chan Event
	es      *fsevents.EventStream
	closed  chan struct{}
	done    chan struct{}
}

func newFSWatcher(root string, selectPath func(path string, info os.FileInfo) bool) (*FSWatcher, error) {
	dev, err := fsevents.DeviceForPath(root)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve device for path %v: %v", root, err)
	}

	es := &fsevents.EventStream{
		Paths:   []string{root},
		Latency: 200 * time.Millisecond,
		Device:  dev,
		Flags:   fsevents.FileEvents | fsevents.WatchRoot,
	}
	es.Start()

	// fsevents returns paths relative to device root so we need
	// to figure out the actual mount point
	mountPoint, err := filepath.Abs(root)
	if err != nil {
		log.Fatal(err)
	}

	for mountPoint != string(os.PathSeparator) {
		parent := filepath.Dir(mountPoint)
		pDev, err := fsevents.DeviceForPath(parent)
		if err != nil {
			log.Fatal(err)
		}
		if pDev != dev {
			break
		}
		mountPoint = parent
	}

	w := &FSWatcher{
		eventCh: make(chan Event),
		es:      es,
		closed:  make(chan struct{}),
		done:    make(chan struct{}),
	}

	go func() {
		defer close(w.eventCh)
		defer close(w.done)
		for {
			select {
			case events := <-es.Events:
				for i := range events {
					event := &events[i]
					log.Printf("got fsevents event at %q; flags %019b", event.Path, event.Flags)
					path := filepath.Join(mountPoint, event.Path)
					info, _ := os.Stat(path)
					if !selectPath(path, info) {
						log.Printf("not selected")
						continue
					}
					log.Printf("selected")

					// Darwin might include both "created" and "changed" in the same event
					// so ordering matters below. The "created" case should be checked
					// before "changed" to get a behavior that is more consistent with other
					// os_others.go.
					switch {
					case event.Flags&fRemoved != 0:
						w.send(Event{path, Removed})
					case event.Flags&fCreated != 0:
						w.send(Event{path, Created})
					case event.Flags&fChanged != 0:
						w.send(Event{path, Changed})
					}
				}
			case <-w.closed:
				return
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

// Close closes the watcher. The caller must still be ready to receive
// events from the Events channel when this is called.
func (w *FSWatcher) Close() error {
	w.es.Stop()
	close(w.closed)
	<-w.done
	return nil
}

// Events returns a channel on which update events can be received.
func (w *FSWatcher) Events() <-chan Event {
	return w.eventCh
}
