//go:build tools
// +build tools

package tools

// This is a kinda hack to ensure that the gopm module has dependencies
// on the Go tools that are used for code generation.

import (
	_ "github.com/golang/protobuf/protoc-gen-go"
)
