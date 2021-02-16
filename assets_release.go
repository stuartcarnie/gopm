// +build release

package gopm

import (
	"embed"
	"io/fs"
	"net/http"

	"go.uber.org/zap"
)

// HTTP auto generated
//go:embed webgui
var content embed.FS

var HTTP http.FileSystem = func() http.FileSystem {
	dir, err := fs.Sub(content, "webgui")
	if err != nil {
		zap.L().Fatal("Failed to load embedded assets.", zap.Error(err))
	}

	return http.FS(dir)
}()
