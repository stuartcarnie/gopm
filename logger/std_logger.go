package logger

import (
	"io"
)

// stdLogger stdout/stderr logger implementation
type stdLogger struct {
	io.Writer
}

// NewStdLogger returns a logger that logs to the given writer.
func NewStdLogger(w io.Writer) Logger {
	return &stdLogger{
		Writer: w,
	}
}

func (l *stdLogger) Close() error {
	return nil
}
