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
	assert.Equal(t, "more\n", string(b.Bytes()))
}

func TestBacklog_NewlineAtEndOfBuffer(t *testing.T) {
	b := NewRingBuffer(10)
	n, err := b.Write([]byte("hello world\nhello world\n"))
	require.NoError(t, err)
	assert.Equal(t, 24, n)
	assert.Equal(t, "", string(b.Bytes()))
}
