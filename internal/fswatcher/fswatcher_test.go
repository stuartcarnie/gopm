package fswatcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestFSWatcher(t *testing.T) {
	dir := t.TempDir()
	path := func(s string) string {
		return filepath.Join(dir, filepath.FromSlash(s))
	}
	mkdir(t, path("d"))
	writefile(t, path("d/f"), "a")
	writefile(t, path("g"), "b")

	w, err := New(dir, nil)
	qt.Assert(t, err, qt.IsNil)

	// No events should happen initially.
	qt.Assert(t, readEvents(w), qt.HasLen, 0)

	writefile(t, path("d/g"), "something")
	qt.Assert(t, readEvents(w), qt.DeepEquals, map[string]Op{
		path("d/g"): Created,
	})
	writefile(t, path("d/f"), "a1")    // change
	writefile(t, path("g"), "b1")      // change
	mkdir(t, path("d/e"))              // create
	writefile(t, path("d/e/p"), "foo") // create
	rm(t, path("d/g"))                 // remove
	qt.Assert(t, readEvents(w), qt.DeepEquals, map[string]Op{
		path("d/e/p"): Created,
		path("d/f"):   Changed,
		path("d/g"):   Removed,
		path("g"):     Changed,
	})

	// Change a file, then close the watcher to check that doesn't deadlock.
	writefile(t, path("g"), "b1") // change
	time.Sleep(time.Millisecond)
	w.Close()
}

func TestFSWatcherWithSelect(t *testing.T) {
	dir := t.TempDir()
	path := func(s string) string {
		return filepath.Join(dir, filepath.FromSlash(s))
	}
	mkdir(t, path("d"))
	writefile(t, path("d/f"), "a")
	writefile(t, path("g"), "b")

	w, err := New(dir, func(s string, info os.FileInfo) bool {
		return info.IsDir() || strings.HasSuffix(s, "x")
	})
	qt.Assert(t, err, qt.IsNil)
	defer w.Close()

	writefile(t, path("f"), "f")
	writefile(t, path("fx"), "")
	mkdir(t, path("dx"))
	writefile(t, path("dx/fx"), "fx")
	mkdir(t, path("d"))
	writefile(t, path("d/g"), "g")
	writefile(t, path("d/gx"), "gx") // Matching file inside non-matching directory.
	qt.Assert(t, readEvents(w), qt.DeepEquals, map[string]Op{
		path("fx"):    Created,
		path("dx/fx"): Created,
		path("d/gx"):  Created,
	})
}

func rm(t *testing.T, path string) {
	err := os.RemoveAll(path)
	qt.Assert(t, err, qt.IsNil)
}

func mkdir(t *testing.T, path string) {
	err := os.MkdirAll(path, 0o777)
	qt.Assert(t, err, qt.IsNil)
}

func writefile(t *testing.T, path, content string) {
	err := os.WriteFile(path, []byte(content), 0o666)
	qt.Assert(t, err, qt.IsNil)
}

func readEvents(w *FSWatcher) map[string]Op {
	es := make(map[string]Op)
loop:
	for {
		select {
		case e, ok := <-w.Events():
			if !ok {
				break loop
			}
			switch {
			case e.Op == Changed && es[e.Path] == Created:
				// Leave the created event there.
			default:
				es[e.Path] = e.Op
			}
		case <-time.After(100 * time.Millisecond):
			break loop
		}
	}
	return es
}
