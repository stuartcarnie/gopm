package process

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rogpeppe/retry"
	"go.uber.org/zap"

	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/logger"
)

func newReadyNotifier(p *process) readyNotifier {
	if p.config.Oneshot {
		// A one-shot process is only ready when it exits,
		// so the associated readyNotifier will never report that
		// it's ready.
		// TODO this is wrong. According the docs, we should
		// run the notifier exactly once after a process has exited,
		// so maybe we can constrain max attempts to 1 in that case
		// and assume that newReadyNotifer is only called after the
		// process has exited.
		return neverReady{}
	}
	if p.config.Probe == nil {
		// No explicit probe, so it's considered ready
		// after some period of time running.
		return newReadyAfter(p.config.StartSeconds.D)
	}
	return newReadyNotifierFromProbe(p, p.config.Probe)
}

func newReadyNotifierFromProbe(p *process, probe *config.Probe) readyNotifier {
	switch t := probe.Type(); t {
	case config.ProbeCommand:
		return newPollingProber(&commandPoller{
			config: p.config,
			probe:  probe,
		}, p.zlog, probe.Retry.Strategy())
	case config.ProbeURL:
		return newPollingProber(&urlPoller{
			url: probe.URL,
		}, p.zlog, probe.Retry.Strategy())
	case config.ProbeFile:
		var pattern *regexp.Regexp
		if probe.Pattern != nil {
			// We need ^ and $ to match newlines within the file, so recompile the
			// pattern with that flag enabled.
			pattern = regexp.MustCompile("(?m)" + probe.Pattern.R.String())
		}
		file := probe.File
		if !filepath.IsAbs(file) {
			file = filepath.Join(p.config.Directory, file)
		}
		// TODO we could be more intelligent about file probe logic:
		// instead of polling, we could set up a watch on the file
		// to notify when the file has changed, and read only
		// data from the last read point in the file (like tail -f).
		return newPollingProber(&filePoller{
			file:    file,
			pattern: pattern,
		}, p.zlog, probe.Retry.Strategy())
	case config.ProbeOutput:
		return newOutputProber(p, probe)
	case config.ProbeAnd:
		return newAndProber(p, probe)
	case config.ProbeOr:
		return newOrProbe(p, probe)
	default:
		panic(fmt.Errorf("unknown probe type %v", t))
	}
}

func newAndProber(p *process, probe *config.Probe) readyNotifier {
	if len(probe.And) == 0 {
		return alwaysReady{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	readyCh := make(chan struct{})
	remain := int64(len(probe.And))
	for _, child := range probe.And {
		child := child
		notifier := newReadyNotifierFromProbe(p, child)
		go func() {
			defer notifier.close()

			select {
			case <-notifier.ready():
				if n := atomic.AddInt64(&remain, -1); n == 0 {
					close(readyCh)
				}
			case <-ctx.Done():
			}
		}()
	}
	return &andProbe{
		cancel:  cancel,
		readyCh: readyCh,
	}
}

type andProbe struct {
	cancel  func()
	readyCh chan struct{}
}

func (p *andProbe) ready() <-chan struct{} {
	return p.readyCh
}

func (p *andProbe) close() {
	p.cancel()
}

func newOrProbe(p *process, probe *config.Probe) readyNotifier {
	if len(probe.Or) == 0 {
		return neverReady{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	var cancelOnce sync.Once
	readyCh := make(chan struct{})
	for _, child := range probe.Or {
		notifier := newReadyNotifierFromProbe(p, child)
		go func() {
			defer notifier.close()
			select {
			case <-notifier.ready():
				cancelOnce.Do(func() {
					cancel()
					close(readyCh)
				})
			case <-ctx.Done():
			}
		}()
	}
	return &orProbe{
		cancel:  cancel,
		readyCh: readyCh,
	}
}

type orProbe struct {
	cancel  func()
	readyCh chan struct{}
}

func (p *orProbe) ready() <-chan struct{} {
	return p.readyCh
}

func (p *orProbe) close() {
	p.cancel()
}

type alwaysReady struct{}

func (alwaysReady) ready() <-chan struct{} {
	return closedChan
}

func (alwaysReady) close() {}

var closedChan = func() <-chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}()

type neverReady struct{}

func (neverReady) ready() <-chan struct{} {
	return nil
}

func (neverReady) close() {}

type readyAfter struct {
	c chan struct{}
	t *time.Timer
}

func newReadyAfter(d time.Duration) *readyAfter {
	c := make(chan struct{})
	var t *time.Timer
	if d > 0 {
		t = time.AfterFunc(d, func() {
			close(c)
		})
	} else {
		close(c)
	}
	return &readyAfter{
		c: c,
		t: t,
	}
}

func (r *readyAfter) ready() <-chan struct{} {
	return r.c
}

func (r *readyAfter) close() {
	if r.t != nil {
		r.t.Stop()
	}
}

type readyNotifier interface {
	ready() <-chan struct{}
	close()
}

type pollingProber struct {
	retryStrategy retry.Strategy
	poller        poller
	readyCh       chan struct{}
	cancel        func()
	zlog          *zap.Logger
}

func newPollingProber(p poller, zlog *zap.Logger, retryStrategy retry.Strategy) *pollingProber {
	ctx, cancel := context.WithCancel(context.Background())
	pp := &pollingProber{
		retryStrategy: retryStrategy,
		poller:        p,
		readyCh:       make(chan struct{}),
		cancel:        cancel,
		zlog:          zlog,
	}
	go pp.run(ctx)
	return pp
}

func (p *pollingProber) ready() <-chan struct{} {
	return p.readyCh
}

func (p *pollingProber) close() {
	p.cancel()
}

func (p *pollingProber) run(ctx context.Context) {
	for i := p.retryStrategy.Start(); ; {
		// Call Next before polling because there's very little chance
		// of a poll succeeding if it's done immediately after the command has started.
		if !i.Next(ctx.Done()) {
			// Note: this can't happen unless the context is done
			// because the MaxCount and MaxDuration fields are always zero currently.
			return
		}
		p.zlog.Debug("running probe poll", zap.Stringer("poller", p.poller))
		ok, err := p.poller.poll(ctx)
		if err != nil {
			if ctx.Err() == nil {
				p.zlog.Error("probe failed", zap.Error(ctx.Err()))
			}
			// TODO What do we do when the readiness notifier fails permanently?
			// (for example because it's waited too long for some status).
			// We'd like to go into Fatal state but then we'd need to kill off the
			// running process, and what's the state until then? Stopping doesn't
			// seem quite right.
			// Currently the process will just stay in Starting state indefinitely.
			return
		}
		if ok {
			close(p.readyCh)
			return
		}
	}
}

type poller interface {
	poll(ctx context.Context) (bool, error)
	String() string
}

type commandPoller struct {
	config *config.Program
	logger *logger.Logger
	probe  *config.Probe
}

func (p *commandPoller) poll(ctx context.Context) (bool, error) {
	if info, err := os.Stat(p.config.Directory); err != nil {
		return false, fmt.Errorf("invalid directory for process %q: %v", p.config.Name, err)
	} else if !info.IsDir() {
		return false, fmt.Errorf("invalid directory for process %q: %q is not a directory", p.config.Name, p.config.Directory)
	}
	cmd := exec.CommandContext(ctx, p.config.Probe.Shell, "-c", p.probe.Command)
	// TODO set user ID
	for k, v := range p.config.Environment {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Dir = p.config.Directory
	cmd.Stdout = p.logger
	cmd.Stderr = p.logger
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if _, ok := err.(*exec.ExitError); !ok {
		// Can't even run the command, so it's not worth retrying.
		return false, err
	}
	return false, nil
}

func (p *commandPoller) String() string {
	return fmt.Sprintf("command %q", p.probe.Command)
}

type filePoller struct {
	file    string
	pattern *regexp.Regexp
}

func (p *filePoller) String() string {
	if p.pattern != nil {
		return fmt.Sprintf("file %q matching %q", p.file, p.pattern)
	}
	return fmt.Sprintf("file %q existence", p.file)
}

func (p *filePoller) poll(ctx context.Context) (bool, error) {
	_, err := os.Stat(p.file)
	if err != nil {
		return false, nil
	}
	if p.pattern == nil {
		return true, nil
	}
	f, err := os.Open(p.file)
	if err != nil {
		return false, err
	}
	defer f.Close()
	return p.pattern.MatchReader(bufio.NewReader(f)), nil
}

type urlPoller struct {
	url string
}

func (p *urlPoller) poll(ctx context.Context) (bool, error) {
	req, err := http.NewRequest("GET", p.url, nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil
	}
	resp.Body.Close()
	return resp.StatusCode/100 == 2, nil
}

func (p *urlPoller) String() string {
	return fmt.Sprintf("HTTP GET %q", p.url)
}

func (p *urlPoller) close() {}

type outputProber struct {
	pattern    *regexp.Regexp
	logger     *logger.Logger
	readyCh    chan struct{}
	buf        []byte
	foundMatch bool
}

func newOutputProber(p *process, probe *config.Probe) *outputProber {
	prober := &outputProber{
		pattern: probe.Output.R,
		logger:  p.logger,
		readyCh: make(chan struct{}),
	}
	p.logger.AddWriter(prober, 0, true)
	return prober
}

func (p *outputProber) ready() <-chan struct{} {
	return p.readyCh
}

func (p *outputProber) close() {
	// Note: don't look at p.foundMatch because
	// close is called concurrently. RemoveWriter
	// is a no-op with the writer isn't present.
	p.logger.RemoveWriter(p)
}

// Write implements io.Writer by triggering the ready
// notification when a match is found.
func (p *outputProber) Write(buf0 []byte) (int, error) {
	if p.foundMatch {
		return len(buf0), nil
	}
	buf := buf0
	for {
		i := bytes.IndexByte(buf, '\n')
		if i == -1 {
			p.buf = append(p.buf, buf...)
			return len(buf0), nil
		}
		line := buf[:i]
		if len(p.buf) > 0 {
			p.buf = append(p.buf, line...)
			line = p.buf
			p.buf = p.buf[:0]
		}
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if p.pattern.Match(line) {
			p.foundMatch = true
			close(p.readyCh)
			return len(buf0), nil
		}
		buf = buf[i+1:]
	}
}

func (p *outputProber) Close() error {
	return nil
}
