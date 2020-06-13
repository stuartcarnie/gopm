package process

import (
	"bytes"
	"sync"
)

// RingBuffer is a ring-buffer.
type RingBuffer struct {
	mu   sync.Mutex
	data []byte
	size int
	w    int
}

// NewBacklog creates a new RingBuffer of a specific size in bytes.
func NewBacklog(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Empty zeroes the buffer.
func (l *RingBuffer) Empty() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.w = 0
	l.data = make([]byte, l.size)
}

// Write writes data to the RingBuffer.
func (l *RingBuffer) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	// If we are writing more bytes than are available in the ring-buffer, skip
	// to the bytes that would not be overwritten by the wrapping behavior.
	tn := len(b)
	if tn > l.size {
		b = b[tn-l.size:]
	}

	// Copy into the buffer
	copy(l.data[l.w:], b)
	left := l.size - l.w
	n := len(b)
	if n > left {
		copy(l.data, b[left:])
	}

	// Advance the write head
	l.w = ((l.w + n) % l.size)

	return tn, nil
}

// Bytes returns a byte slice containing all data written to the buffer. We trim
// the output to the first newline character (if we can find one) in order to
// keep the output looking nice.
func (l *RingBuffer) Bytes() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]byte, l.size)
	copy(out, l.data[l.w:])
	copy(out[l.size-l.w:], l.data[:l.w])

	// Trim the bit before the first newline
	idx := bytes.IndexRune(out, '\n')
	if idx > -1 {
		return out[idx+1:]
	}
	return out
}
