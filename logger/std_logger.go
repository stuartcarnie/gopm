package logger

import (
	"io"
	"os"
)

// StdLogger stdout/stderr logger implementation
type StdLogger struct {
	io.Writer
	// Add an extra level to the NullLogger methods so
	// that Writer.Write overwrites NullLogger.Write.
	nullLogger
}

type nullLogger struct {
	NullLogger
}

// NewStdLogger returns a logger that logs to the given writer.
func NewStdLogger(w io.Writer) *StdLogger {
	return &StdLogger{
		Writer: w,
	}
}

// NewStdoutLogger returns a logger that logs to os.Stdout.
func NewStdoutLogger() *StdLogger {
	return NewStdLogger(os.Stdout)
}

// NewStderrLogger returns a logger that logs to os.Stderr.
func NewStderrLogger() *StdLogger {
	return NewStdLogger(os.Stderr)
}
