package gopm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLastNLines(t *testing.T) {
	content := []byte("hello\nworld\nand\ngopm\n")

	lines := lastNLines(content, 2)
	assert.Equal(t, []byte("and\ngopm\n"), lines)

	lines = lastNLines(content, 3)
	assert.Equal(t, []byte("world\nand\ngopm\n"), lines)

	lines = lastNLines(content, 1)
	assert.Equal(t, []byte("gopm\n"), lines)

	lines = lastNLines(content, 0)
	assert.Equal(t, []byte(""), lines)
}
