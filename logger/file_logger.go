package logger

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
)

// fileLogger log program stdout/stderr to file
type fileLogger struct {
	baseName string
	maxSize  int64
	backups  int
	fileSize int64
	file     *os.File
}

// newFileLogger returns a logger that logs to the file with the given name
// limiting file size to about maxSize bytes and retaining the given
// maximum number of backups.
func newFileLogger(name string, maxSize int64, backups int) (io.WriteCloser, error) {
	logger := &fileLogger{
		baseName: name,
		maxSize:  maxSize,
		backups:  backups,
	}
	if err := logger.openFile(false); err != nil {
		return nil, err
	}
	return logger, nil
}

// open the file and truncate the file if trunc is true
func (l *fileLogger) openFile(trunc bool) error {
	name := l.name(0)
	fileInfo, err := os.Stat(name)
	if trunc || err != nil {
		l.file, err = os.Create(name)
	} else {
		l.fileSize = fileInfo.Size()
		l.file, err = os.OpenFile(name, os.O_RDWR|os.O_APPEND, 0o666)
	}
	return err
}

func (l *fileLogger) backupFiles() {
	for i := l.backups - 1; i >= 0; i-- {
		if _, err := os.Stat(l.name(i)); err == nil {
			if err := os.Rename(l.name(i), l.name(i+1)); err != nil {
				zap.L().Error("cannot rename backup log file", zap.Error(err), zap.String("from", l.name(i)), zap.String("to", l.name(i+1)))
			}
		}
	}
}

func (l *fileLogger) name(n int) string {
	if n == 0 {
		return l.baseName
	}
	return fmt.Sprintf("%s.%d", l.baseName, n)
}

// Write Override the function in io.Writer. Write the log message to the file
func (l *fileLogger) Write(p []byte) (int, error) {
	if l.file == nil {
		return 0, fmt.Errorf("log file is moribund")
	}
	n, err := l.file.Write(p)
	l.fileSize += int64(n)
	if l.fileSize >= l.maxSize {
		l.Close()
		l.backupFiles()
		if err := l.openFile(true); err != nil {
			zap.L().Error("cannot open fresh log file", zap.Error(err))
		}
		l.fileSize = 0
	}
	return n, err
}

// Close closes the file logger
func (l *fileLogger) Close() error {
	if file := l.file; file != nil {
		l.file = nil
		return file.Close()
	}
	return nil
}
