package model

import (
	"io"

	"github.com/goccy/go-yaml"
)

func ParseRoot(reader io.Reader) (*Root, error) {
	dec := yaml.NewDecoder(reader)
	var v Root
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}
