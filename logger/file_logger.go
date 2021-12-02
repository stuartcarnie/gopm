package logger

import (
	"fmt"
	"os"
	"sync"
)

// fileLogger log program stdout/stderr to file
type fileLogger struct {
	name     string
	maxSize  int64
	backups  int
	fileSize int64
	file     *os.File
	locker   sync.Locker
}

// newFileLogger returns a logger that logs to the file with the given name.
func newFileLogger(name string, maxSize int64, backups int) Logger {
	logger := &fileLogger{
		name:     name,
		maxSize:  maxSize,
		backups:  backups,
		fileSize: 0,
		file:     nil,
	}
	logger.openFile(false)
	return logger
}

// open the file and truncate the file if trunc is true
func (l *fileLogger) openFile(trunc bool) error {
	if l.file != nil {
		l.file.Close()
	}
	var err error
	fileInfo, err := os.Stat(l.name)

	if trunc || err != nil {
		l.file, err = os.Create(l.name)
	} else {
		l.fileSize = fileInfo.Size()
		l.file, err = os.OpenFile(l.name, os.O_RDWR|os.O_APPEND, 0o666)
	}
	if err != nil {
		fmt.Printf("Fail to open log file --%s-- with error %v\n", l.name, err)
	}
	return err
}

func (l *fileLogger) backupFiles() {
	for i := l.backups - 1; i > 0; i-- {
		src := fmt.Sprintf("%s.%d", l.name, i)
		dest := fmt.Sprintf("%s.%d", l.name, i+1)
		if _, err := os.Stat(src); err == nil {
			os.Rename(src, dest)
		}
	}
	dest := fmt.Sprintf("%s.1", l.name)
	os.Rename(l.name, dest)
}

// Write Override the function in io.Writer. Write the log message to the file
func (l *fileLogger) Write(p []byte) (int, error) {
	l.locker.Lock()
	defer l.locker.Unlock()

	n, err := l.file.Write(p)
	if err != nil {
		return n, err
	}
	l.fileSize += int64(n)
	if l.fileSize >= l.maxSize {
		fileInfo, errStat := os.Stat(l.name)
		if errStat == nil {
			l.fileSize = fileInfo.Size()
		} else {
			return n, errStat
		}
	}
	if l.fileSize >= l.maxSize {
		l.Close()
		l.backupFiles()
		l.openFile(true)
	}
	return n, err
}

// Close close the file logger
func (l *fileLogger) Close() error {
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}
