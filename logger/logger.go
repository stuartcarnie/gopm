package logger

import (
	"io"
	"strings"
)

// Logger the log interface to log program stdout/stderr logs to file
type Logger interface {
	io.WriteCloser
	ReadLog(offset, length int64) (string, error)
	ReadTailLog(offset, length int64) (string, int64, bool, error)
	ClearCurLogFile() error
	ClearAllLogFile() error
}

// NewLogger create a logger for a program with parameters
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
		return NewStdoutLogger()
	case "/dev/stderr":
		return NewStderrLogger()
	case "/dev/null":
		return NewNullLogger()
	default:
		if len(logFile) > 0 {
			return NewFileLogger(logFile, maxBytes, backups)
		}
		return NewNullLogger()
	}
}
