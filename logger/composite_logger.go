package logger

import "sync"

// CompositeLogger dispatch the log message to other loggers
type CompositeLogger struct {
	mu      sync.Mutex
	loggers []Logger
}

// NewCompositeLogger returns a Logger implementation that logs to
// all the given loggers, and also allows loggers to be dynamically added
// and removed. Logger implementations must be comparable.
//
// The first logger is special: Write and Close errors from all but the first logger are ignored.
func NewCompositeLogger(loggers ...Logger) *CompositeLogger {
	return &CompositeLogger{loggers: loggers}
}

// AddLogger add a logger to receive the log data
func (cl *CompositeLogger) AddLogger(logger Logger) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.loggers = append(cl.loggers, logger)
}

// RemoveLogger removes the given logger from the loggers in cl.
func (cl *CompositeLogger) RemoveLogger(logger Logger) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	for i, t := range cl.loggers {
		if t == logger {
			cl.loggers = append(cl.loggers[:i], cl.loggers[i+1:]...)
			break
		}
	}
}

// Write implements Logger.Write by writing to all the loggers in cl.
func (cl *CompositeLogger) Write(p []byte) (n int, err error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	for i, logger := range cl.loggers {
		if i == 0 {
			n, err = logger.Write(p)
		} else {
			logger.Write(p)
		}
	}
	return
}

// Close closes all the loggers in CL.
func (cl *CompositeLogger) Close() (err error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	for i, logger := range cl.loggers {
		if i == 0 {
			err = logger.Close()
		} else {
			logger.Close()
		}
	}
	return
}
