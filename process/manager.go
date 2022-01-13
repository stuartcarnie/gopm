package process

import (
	"context"
	"errors"
	"io"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/logger"
	"github.com/stuartcarnie/gopm/rpc"
)

// ErrNotFound is returned when a multi-process operation is
// executed but no processes were found to operate on.
var ErrNotFound = errors.New("no matching process found")

// Manager manages all the process in the supervisor.
type Manager struct {
	// notifier is used by processes to notify the world in general
	// about state changes without risk of deadlock.
	notifier *stateNotifier

	// reqc is used to request a configuration change.
	updateConfigc chan *updateConfigReq

	//  mu guards the fields below it. The only place that acquires a read-write
	// lock is the updater when it modifies the current set of processes.
	mu sync.RWMutex

	// processes holds the current set of processes, keyed by name.
	processes map[string]*process

	// config holds the current configuration. There's a one-to-one
	// relationship between processes and config.Programs.
	// This information is also stored in each process, but we
	// avoid accessing that to avoid potential race conditions
	// when the process itself might be updating it.
	config *config.Config
}

func NewManager() *Manager {
	m := &Manager{
		notifier:      newStateNotifier(),
		processes:     make(map[string]*process),
		updateConfigc: make(chan *updateConfigReq),
		config:        new(config.Config),
	}
	go m.updateProc()
	return m
}

type ProcessInfo struct {
	Name        string
	Description string
	Start       time.Time
	Stop        time.Time
	State       State
	// TODO make this an error instead?
	ExitStatus int
	Logfile    string
	Pid        int
}

// AllProcessInfo returns information on all the processes.
func (pm *Manager) AllProcessInfo() []*ProcessInfo {
	zap.L().Debug("Manager.AllProcessInfo")
	reply := make(chan *ProcessInfo, 1)
	n := len(pm.send(processRequest{
		kind:      reqInfo,
		infoReply: reply,
	}, matchAll))
	infos := make([]*ProcessInfo, 0, n)
	for i := 0; i < n; i++ {
		infos = append(infos, <-reply)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos
}

// SignalProcesses sends the given signal to all matching processes.
func (pm *Manager) SignalProcesses(name string, labels map[string]string, sig config.Signal) error {
	zap.L().Debug("Manager.SignalProcesses", zap.String("name", name), zap.Any("labels", labels), zap.Stringer("signal", sig))
	procs := pm.send(processRequest{
		kind:   reqSignal,
		signal: sig,
	}, programMatcher(name, labels))
	if len(procs) == 0 {
		return ErrNotFound
	}
	// We could potentially wait here until the signal(s) have actually been sent,
	// but is that worth doing, given that signal delivery is essentially asynchronous
	// anyway?
	return nil
}

// StartAllProcesses starts all the processes. If some failed to start, it
// returns an *rpc.NotStartedError describing the processes that didn't.
func (pm *Manager) StartAllProcesses() error {
	zap.L().Debug("Manager.StartAllProcesses")
	procs := pm.send(processRequest{
		kind:         reqStart,
		stateUpdated: make(chan struct{}, 1),
	}, matchAll)

	return maybeNotReadyError(<-pm.notifier.waitFor(nil, procs, isReady))
}

// StartProcesses starts all matching processes and their dependencies.
// If some failed to start, it
// returns an *rpc.NotStartedError describing the processes that didn't.
func (pm *Manager) StartProcesses(name string, labels map[string]string) error {
	zap.L().Debug("Manager.StartProcesses", zap.String("name", name), zap.Any("labels", labels))
	return pm.startProcesses(context.Background(), programMatcher(name, labels))
}

// startProcesses is the internal version of StartProcesses, also called by RestartProcesses.
func (pm *Manager) startProcesses(ctx context.Context, match func(*config.Program) bool) error {
	pm.mu.RLock()
	deps := getDeps(pm.config, match)

	procs := pm._send(processRequest{
		kind:         reqStart,
		stateUpdated: make(chan struct{}, 1),
	}, func(p *config.Program) bool {
		return deps[p.Name]&(depTarget|depDependedOn) != 0
	})
	if len(procs) == 0 {
		pm.mu.RUnlock()
		return ErrNotFound
	}
	// Resume any processes that depend on the target set but
	// were only paused. We won't wait for them.
	pm._send(processRequest{
		kind: reqResume,
	}, func(p *config.Program) bool {
		return deps[p.Name] == depDependsOn
	})
	pm.mu.RUnlock()
	return maybeNotReadyError(<-pm.notifier.waitFor(nil, procs, isReady))
}

// StopAllProcesses stops all the processes managed by this manager
func (pm *Manager) StopAllProcesses() {
	zap.L().Debug("Manager.StopAllProcesses")
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pm.stopOrPauseProcesses(context.Background(), func(p *config.Program) bool {
		return true
	}, reqStop)
}

// StopProcesses stops all matching processes.
func (pm *Manager) StopProcesses(name string, labels map[string]string) error {
	zap.L().Debug("Manager.StopProcesses", zap.String("name", name), zap.Any("labels", labels))
	return pm.stopProcesses(context.Background(), programMatcher(name, labels))
}

// stopProcesses is the internal version of StopProcesses. It's also invoked as part of RestartProcesses.
func (pm *Manager) stopProcesses(ctx context.Context, match func(*config.Program) bool) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	stopped, err := pm.stopOrPauseProcesses(context.Background(), match, reqStop)
	if err != nil {
		return err
	}
	if len(stopped) == 0 {
		return ErrNotFound
	}
	return nil
}

// RestartProcesses restarts all matching processes by stopping them, then starting
// them again.
func (pm *Manager) RestartProcesses(name string, labels map[string]string) error {
	zap.L().Debug("Manager.RestartProcesses", zap.String("name", name), zap.Any("labels", labels))
	ctx := context.Background()
	match := programMatcher(name, labels)
	if err := pm.stopProcesses(ctx, match); err != nil {
		return err
	}
	return pm.startProcesses(ctx, match)
}

func maybeNotReadyError(notReadyProcs []*process) error {
	if len(notReadyProcs) == 0 {
		return nil
	}
	return &rpc.NotStartedError{
		ProcessNames: procNames(notReadyProcs),
	}
}

func procNames(procs []*process) []string {
	names := make([]string, len(procs))
	for i, p := range procs {
		names[i] = p.name
	}
	return names
}

type TailLogParams struct {
	Name         string
	BacklogLines int64
	Follow       bool
	Writer       io.Writer
}

// TailLog tails the log of process named by p.Name, calling
// p.Write when data is received. When pm.Follow is true,
// the log will continue to be written until the context is cancelled.
func (pm *Manager) TailLog(ctx context.Context, p TailLogParams) error {
	reply := make(chan *logger.Logger, 1)
	procs := pm.send(processRequest{
		kind:        reqLogger,
		loggerReply: reply,
	}, programMatcher(p.Name, nil))
	if len(procs) == 0 {
		return ErrNotFound
	}
	logger := <-reply
	closed := make(chan struct{})
	w := &loggerWriter{
		Writer: p.Writer,
		closed: closed,
	}
	logger.AddWriter(w, p.BacklogLines, p.Follow)
	defer logger.RemoveWriter(w)
	select {
	case <-ctx.Done():
	case <-closed:
	}
	return nil
}

type loggerWriter struct {
	io.Writer
	closed chan struct{}
}

func (w *loggerWriter) Close() error {
	if w.closed != nil {
		close(w.closed)
		w.closed = nil
	}
	return nil
}

// send sends the given request to all processes for which match returns true.
//
// It returns the processes that were sent the request.
func (pm *Manager) send(req processRequest, match func(p *config.Program) bool) []*process {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm._send(req, match)
}

// _send is like send but doesn't acquire pm.mu.
func (pm *Manager) _send(req processRequest, match func(p *config.Program) bool) []*process {
	procs := make([]*process, 0, len(pm.processes))
	for _, p := range pm.config.Programs {
		if match(p) {
			proc := pm.processes[p.Name]
			proc.req <- req
			procs = append(procs, proc)
		}
	}
	if req.stateUpdated != nil {
		for range procs {
			<-req.stateUpdated
		}
	}
	return procs
}

func matchAll(*config.Program) bool {
	return true
}

func programMatcher(name string, labels map[string]string) func(*config.Program) bool {
	return func(p *config.Program) bool {
		return matchProgram(p, name, labels)
	}
}

func matchProgram(p *config.Program, name string, labels map[string]string) bool {
	if name != "" && p.Name != name {
		return false
	}
	for k, v := range labels {
		pv, ok := p.Labels[k]
		if !ok || v != pv {
			return false
		}
	}
	return true
}
