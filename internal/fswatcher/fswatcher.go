// This code was adapted from the github.com/govim/govim/cmd/govim/internal/fswatcher
// package at commit 595708280f738dcb39176591574559a308631dde.

// Package fswatcher is responsible for providing file system events to govim
package fswatcher

import (
	"fmt"
	"os"
)

// New creates a file watcher that provides events recursively for all files and
// directories with paths p and file stat information info such that selectPath(p, info) returns true.
// The info argument passed to selectPath may be nil (for example when a file is removed).
//
// If a directory is not selected, its descendants may or not be presented as candidates for selection
// too, depending on the platform, so selectPath should be written to take that into account.
//
// The root argument must be the path to a directory.
//
// The returned watcher should be closed by calling Close after use.
//
// If selectPath is nil, all paths will be selected.
func New(root string, selectPath func(path string, info os.FileInfo) bool) (*FSWatcher, error) {
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("provided root %q must be an existing directory", root)
	}
	if selectPath == nil {
		selectPath = func(string, os.FileInfo) bool {
			return true
		}
	}
	return newFSWatcher(root, selectPath)
}

// Event represents
type Event struct {
	Path string
	Op   Op
}

func (e Event) String() string {
	return fmt.Sprintf("%v %q", e.Op, e.Path)
}

//go:generate stringer -type Op

// Op represents a kind of event that can happen within a directory.
type Op int

const (
	_ = Op(iota)
	Created
	Changed
	Removed
)
