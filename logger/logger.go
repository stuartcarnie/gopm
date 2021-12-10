package logger

import (
	"fmt"
	"io"
	"os"
	"sync"

	"go.uber.org/zap"
)

type Logger struct {
	mu      sync.Mutex
	writers []io.WriteCloser
	buf     ringBuffer
	closed  bool
}

// New returns a logger that writes logs to the given file (if logFile is non-empty),
// keeping log files near to the given maximum file size
// and using a maximum of maxBacklog bytes of memory for the
// backlog. Up to the given number of backup files (minimum 1)
// will be retained.
func New(logFile string, maxFileSize int64, maxBacklog int, backups int) *Logger {
	l := &Logger{
		buf: ringBuffer{
			maxSize: maxBacklog,
		},
	}
	var w io.WriteCloser
	switch logFile {
	case "/dev/stdout":
		w = nopCloser{os.Stdout}
	case "/dev/stderr":
		w = nopCloser{os.Stderr}
	case "/dev/null", "":
		// Don't log at all.
	default:
		w1, err := newFileLogger(logFile, maxFileSize, backups)
		if err != nil {
			zap.L().Error("cannot create logger", zap.Error(err))
		} else {
			w = w1
		}
	}
	if w != nil {
		l.writers = append(l.writers, w)
	}
	return l
}

func (l *Logger) Write(buf []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.writers {
		if _, err := w.Write(buf); err != nil {
			zap.L().Error("error writing to log file", zap.Error(err))
		}
	}
	l.buf.Write(buf)
	return len(buf), nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.writers {
		w.Close()
	}
	l.writers = nil
	l.closed = true
	return nil
}

// AddWriter adds the given writer, which should be comparable,
//  to the logger. It starts by writing
// at most backlogLines of previous output and then, if follow is true,
// continues writing output as it arrives.
//
// The writer is closed when there's nothing more
// to write (when follow is false and we've got to the end of the log
// or when the logger itself has been closed).
//
// The writer should be removed after use with RemoveWriter.
func (l *Logger) AddWriter(w io.WriteCloser, backlogLines int64, follow bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed || (backlogLines <= 0 && !follow) {
		w.Close()
		return
	}
	if backlogLines > 0 {
		// Start a goroutine to send the backlog so reading
		// a very long backlog doesn't get in the way of producing
		// new log data.
		t := &tailer{
			logger:       l,
			w:            w,
			backlogLines: backlogLines,
			follow:       follow,
		}
		go t.run()
		w = t
	}
	l.writers = append(l.writers, w)
}

type tailer struct {
	backlogLines int64
	follow       bool
	logger       *Logger
	w            io.WriteCloser

	mu     sync.Mutex
	buf    []byte
	closed bool
}

// Write records the data so that it's not lost while the
// tailer is sending its backlog.
func (t *tailer) Write(buf []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return 0, fmt.Errorf("tailer logger has been closed")
	}
	t.buf = append(t.buf, buf...)
	return 0, nil
}

func (t *tailer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return t.w.Close()
}

func (t *tailer) run() {
	// TODO potentially this could look in file data too
	// when we're asking for more lines than there are
	// available.
	_, backlog := t.logger.buf.bytes()
	backlog = lastNLines(backlog, int(t.backlogLines))
	for {
		if _, err := t.w.Write(backlog); err != nil {
			t.w.Close()
			t.logger.RemoveWriter(t)
			return
		}
		// Write any backlog that's accumulated since.
		t.mu.Lock()
		backlog := t.buf
		t.buf = nil
		t.mu.Unlock()
		if t.closed {
			return
		}
		if len(backlog) == 0 {
			break
		}
	}
	// Note: obtaining a lock on t.logger prevents t.Write being
	// called, so we know we can manipulate the set of loggers
	// and that t.buf won't change underfoot.
	t.logger.mu.Lock()
	defer t.logger.mu.Unlock()
	if len(t.buf) > 0 {
		if _, err := t.w.Write(t.buf); err != nil {
			t.w.Close()
			t.logger.replace(t, nil)
			return
		}
	}
	if t.follow {
		t.logger.replace(t, t.w)
	} else {
		t.w.Close()
		t.logger.replace(t, nil)
	}
}

// RemoveWriter removes w from the set of writers being logged to.
func (l *Logger) RemoveWriter(w io.WriteCloser) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.replace(w, nil)
}

// replace replaces the logger w with a replacement writer.
// If with is nil, w is removed.
func (l *Logger) replace(w io.WriteCloser, with io.WriteCloser) {
	for i, lw := range l.writers {
		if !isWriter(lw, w) {
			continue
		}
		if with != nil {
			l.writers[i] = with
			return
		}
		l.writers[i] = nil
		n := len(l.writers)
		if i < n-1 {
			l.writers[i] = l.writers[n-1]
			l.writers[n-1] = nil
		}
		l.writers = l.writers[:n-1]
		break
	}
}

// isWriter reports whether lw is the same as w
// or a tailer instance holding it.
func isWriter(lw, w io.WriteCloser) bool {
	if lw == w {
		return true
	}
	if lw, ok := lw.(*tailer); ok {
		return lw.w == w
	}
	return false
}

type nopCloser struct {
	io.Writer
}

func (c nopCloser) Close() error {
	return nil
}
