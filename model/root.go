package model

import "github.com/stuartcarnie/gopm/pkg/env"

type Root struct {
	Environment env.KeyValues `yaml:"environment"`
	HttpServer  *HTTPServer   `yaml:"http_server"`
	GrpcServer  *GrpcServer   `yaml:"grpc_server"`
	Programs    []*Program    `yaml:"programs"`
	Groups      []*Group      `yaml:"groups"`
	FileSystem  *FileSystem
}

type FileSystem struct {
	Root  string
	Files []*File
}

type File struct {
	Name    string
	Path    string
	Content string
}
