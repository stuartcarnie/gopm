package logger

import (
	"io"
	"os"
	"strings"
)

// Logger the log interface to log program stdout/stderr logs to file
type Logger interface {
	io.WriteCloser
}

// NewLogger returns a logger for a program with the given name,
// writing logs to any log files specified in logFile (each file is comma-separated),
// storing up to maxBytes in each file.
func NewLogger(programName, logFile string, maxBytes int64, backups int) Logger {
	files := splitLogFile(logFile)
	loggers := make([]Logger, 0)
	for _, f := range files {
		loggers = append(loggers, createLogger(programName, f, maxBytes, backups))
	}
	return NewCompositeLogger(loggers...)
}

func splitLogFile(logFile string) []string {
	files := strings.Split(logFile, ",")
	for i, f := range files {
		files[i] = strings.TrimSpace(f)
	}
	return files
}

func createLogger(programName, logFile string, maxBytes int64, backups int) Logger {
	switch logFile {
	case "/dev/stdout":
		return NewStdLogger(os.Stdout)
	case "/dev/stderr":
		return NewStdLogger(os.Stderr)
	case "/dev/null":
		return nullLogger{}
	default:
		if len(logFile) > 0 {
			return newFileLogger(logFile, maxBytes, backups)
		}
		return nullLogger{}
	}
}
