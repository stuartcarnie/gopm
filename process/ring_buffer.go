package process

import (
	"bytes"
	"sync"
)

// ringBuffer is a ring-buffer.
type ringBuffer struct {
	mu      sync.Mutex
	data    []byte
	maxSize int
	p       int64
}

// newRingBuffer creates a new ringBuffer with the given
// maximum size in bytes.
func newRingBuffer(maxSize int) *ringBuffer {
	return &ringBuffer{
		maxSize: maxSize,
	}
}

// Empty zeroes the buffer.
func (l *ringBuffer) Empty() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.p = 0
	l.data = l.data[:0]
}

// Write writes data to the ringBuffer.
func (l *ringBuffer) Write(b []byte) (int, error) {
	nw := len(b)
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.data) < l.maxSize {
		// The buffer hasn't yet grown to full size.
		total := len(l.data) + len(b)
		if total <= l.maxSize {
			// The data fits in the buffer.
			l.data = append(l.data, b...)
			l.p += int64(len(b))
			return nw, nil
		}
		// The data doesn't fit in the buffer, so write what we
		// can and fall through to the wrapping code below.
		cutoff := l.maxSize - len(l.data)
		l.data = append(l.data, b[:cutoff]...)
		l.p = int64(l.maxSize)
		b = b[cutoff:]
	}
	// Buffer overflowed; wrap around in existing buffer.
	if len(b) > l.maxSize {
		// We are writing more bytes than are available in the ring-buffer, so skip
		// the bytes that would be overwritten by the wrapping behavior.
		skipped := len(b) - l.maxSize
		b = b[skipped:]
		l.p += int64(skipped)
	}
	w := int(l.p % int64(l.maxSize))
	remain := copy(l.data[w:], b)
	copy(l.data, b[remain:])
	l.p += int64(len(b))
	return nw, nil
}

// Bytes returns a byte slice containing all data written to the buffer
// and the number of bytes before it that have been discarded because of
// the buffer's size limit.
//
// We trim the output to the first newline character (if we can find
// one) in order to keep the output looking nice.
func (l *ringBuffer) Bytes() (int64, []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.data) < l.maxSize {
		return l.p, append([]byte(nil), l.data...)
	}
	out := make([]byte, 0, l.maxSize)
	w := int(l.p % int64(l.maxSize))
	out = append(out, l.data[w:]...)
	out = append(out, l.data[0:w]...)

	if l.p == 0 {
		return 0, out
	}
	p0 := l.p - int64(l.maxSize)
	// The buffer has wrapped; trim the data before the
	// first newline so we don't see a partial line.
	if idx := bytes.IndexByte(out, '\n'); idx >= 0 {
		return p0 + int64(idx+1), out[idx+1:]
	}
	return p0, out
}
