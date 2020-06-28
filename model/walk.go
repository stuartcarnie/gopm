package model

import (
	"fmt"

	"github.com/stuartcarnie/gopm/pkg/env"
)

type Node interface{}

type Visitor interface {
	Visit(Node) Visitor
}

func Walk(v Visitor, node Node) {
	if v = v.Visit(node); v == nil {
		return
	}

	switch n := node.(type) {
	case *Root:
		for i := range n.Environment {
			Walk(v, &n.Environment[i])
		}

		if n.HttpServer != nil {
			Walk(v, n.HttpServer)
		}

		if n.GrpcServer != nil {
			Walk(v, n.GrpcServer)
		}

		for _, p := range n.Programs {
			Walk(v, p)
		}

		for _, g := range n.Groups {
			Walk(v, g)
		}

		if n.FileSystem != nil {
			Walk(v, n.FileSystem)
		}

	case *FileSystem:
		for _, f := range n.Files {
			Walk(v, f)
		}

	case *Program:
		for i := range n.Environment {
			Walk(v, &n.Environment[i])
		}

	case *env.KeyValues, *env.KeyValue, *HTTPServer, *GrpcServer, *Group, *File:
		// nothing further

	default:
		panic(fmt.Sprintf("model.Walk: unexpected node type %T", n))
	}

	v.Visit(nil)
}

type WalkFunc func(Node) bool

func (fn WalkFunc) Visit(node Node) Visitor {
	if fn(node) {
		return fn
	}
	return nil
}
