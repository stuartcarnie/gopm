package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFilesMissingRootDir(t *testing.T) {
	dir := t.TempDir()
	rootDir := filepath.Join(dir, "/etc/foo")
	files, err := writeFiles(rootDir, nil)
	assert.NoError(t, err)
	assert.Len(t, files, 0)
	info, err := os.Stat(rootDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestWriteFilesOkWithExistingRootDir(t *testing.T) {
	dir := t.TempDir()
	files, err := writeFiles(dir, nil)
	assert.NoError(t, err)
	assert.Len(t, files, 0)
}

func TestWriteFilesDoesNotRewriteMatchingContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	files, err := writeFiles(dir, []*file{{
		Name:    "x",
		Path:    "f",
		Content: "a",
	}})
	require.NoError(t, err)
	path := filepath.Join(dir, "f")

	assert.Equal(t, []*localFile{{
		name:     "x",
		fullPath: path,
	}}, files)
	data, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "a", string(data))

	// Check that the file doesn't get written again by making sure
	// its mtime doesn't change.
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.False(t, info.IsDir())

	files1, err := writeFiles(dir, []*file{{
		Name:    "x",
		Path:    "f",
		Content: "a",
	}})
	require.NoError(t, err)
	assert.Equal(t, files, files1)

	// mtime resolution is often a second, so wait at least that long.
	time.Sleep(time.Second)
	data, err = ioutil.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "a", string(data))

	info1, err := os.Stat(path)
	require.NoError(t, err)
	require.False(t, info.IsDir())
	require.Equal(t, info.ModTime(), info1.ModTime())
}
