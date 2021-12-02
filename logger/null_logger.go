package logger

// NullLogger discard the program stdout/stderr log
type nullLogger struct{}

// Write write the log to this logger
func (l nullLogger) Write(p []byte) (int, error) {
	return len(p), nil
}

// Close close the logger
func (l nullLogger) Close() error {
	return nil
}
