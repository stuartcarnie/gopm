package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBacklog_MultipleWrites(t *testing.T) {
	b := NewRingBuffer(8)
	n, err := b.Write([]byte("hello world\n"))
	require.NoError(t, err)
	assert.Equal(t, 12, n)

	n, err = b.Write([]byte("more\n"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	p, data := b.Bytes()
	assert.Equal(t, p, int64(12))
	assert.Equal(t, "more\n", string(data))
}

func TestBacklog_NewlineAtEndOfBuffer(t *testing.T) {
	b := NewRingBuffer(10)
	content := "hello world\nhello world\n"
	n, err := b.Write([]byte(content))
	require.NoError(t, err)
	assert.Equal(t, 24, n)
	p, data := b.Bytes()
	assert.Equal(t, int64(len(content)), p)
	assert.Equal(t, "", string(data))
}
