package process

import (
	"context"
	"errors"
	"io"
	"sort"
	"sync"
	"time"

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
	reply := make(chan *ProcessInfo)
	n := len(pm.sendAll(processRequest{
		kind:      reqInfo,
		infoReply: reply,
	}))
	infos := make([]*ProcessInfo, 0, n)
	for i := 0; i < n; i++ {
		infos = append(infos, <-reply)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos
}

// RestartProcesses restarts all matching processes. If some failed to restart, it
// returns an *rpc.NotStartedError describing the processes that didn't.
func (pm *Manager) RestartProcesses(name string, labels map[string]string) error {
	procs := pm.send(processRequest{
		kind: reqRestart,
	}, name, labels)
	if len(procs) == 0 {
		return ErrNotFound
	}
	<-pm.notifier.watch(nil, procs, isReady)
	return pm.checkReady(procs)
}

// SignalProcesses sends the given signal to all matching processes.
func (pm *Manager) SignalProcesses(name string, labels map[string]string, sig config.Signal) error {
	procs := pm.send(processRequest{
		kind:   reqSignal,
		signal: sig,
	}, name, labels)
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
	procs := pm.sendAll(processRequest{
		kind: reqStart,
	})
	<-pm.notifier.watch(nil, procs, isReadyOrFailed)
	return pm.checkReady(procs)
}

// StartProcesses starts all matching processes. If some failed to start, it
// returns an *rpc.NotStartedError describing the processes that didn't.
func (pm *Manager) StartProcesses(name string, labels map[string]string) error {
	procs := pm.send(processRequest{
		kind: reqStart,
	}, name, labels)
	if len(procs) == 0 {
		return ErrNotFound
	}
	<-pm.notifier.watch(nil, procs, isReadyOrFailed)
	return pm.checkReady(procs)
}

// StopAllProcesses stops all the processes managed by this manager
func (pm *Manager) StopAllProcesses() {
	procs := pm.sendAll(processRequest{
		kind: reqStop,
	})
	<-pm.notifier.watch(nil, procs, isStopped)
}

// StopProcesses stops all matching processes.
func (pm *Manager) StopProcesses(name string, labels map[string]string) error {
	procs := pm.send(processRequest{
		kind: reqStop,
	}, name, labels)
	if len(procs) == 0 {
		return ErrNotFound
	}
	<-pm.notifier.watch(nil, procs, isStopped)
	return nil
}

func (pm *Manager) checkReady(procs []*process) error {
	if failedProcs := pm.notifier.check(procs, isReady); len(failedProcs) > 0 {
		return &rpc.NotStartedError{
			ProcessNames: procNames(failedProcs),
		}
	}
	return nil
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
	reply := make(chan *logger.Logger)
	procs := pm.send(processRequest{
		kind:        reqLogger,
		loggerReply: reply,
	}, p.Name, nil)
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

// sendAll is like send but sends the request to all processes.
func (pm *Manager) sendAll(req processRequest) []*process {
	return pm.send(req, "", nil)
}

// send sends the given request to all processes with the given name
// (if name is non-empty) and all the given labels.
//
// sendRequest("", nil) will send the request to all processes.
//
// It returns the processes that were sent the request.
func (pm *Manager) send(req processRequest, name string, labels map[string]string) []*process {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	procs := make([]*process, 0, len(pm.processes))
	for _, p := range pm.processes {
		if pm.match(p, name, labels) {
			p.req <- req
			procs = append(procs, p)
		}
	}
	return procs
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

// match reports whether the p matches the given name and labels.
func (pm *Manager) match(p *process, name string, labels map[string]string) bool {
	return matchProgram(pm.config.Programs[p.name], name, labels)
}
