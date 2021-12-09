package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/logger"
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
	ExitStatus  int
	Logfile     string
	Pid         int
}

// AllProcessInfo returns information on all the processes.
func (pm *Manager) AllProcessInfo() []*ProcessInfo {
	// TODO guard against shutdown.
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

// RestartProcesses restarts all matching processes.
func (pm *Manager) RestartProcesses(name string, labels map[string]string) error {
	return fmt.Errorf("unimplemented")
}

// SignalProcesses sends the given signal to all matching processes.
func (pm *Manager) SignalProcesses(name string, labels map[string]string, sig os.Signal) error {
	return fmt.Errorf("unimplemented")
}

// StartAllProcesses starts all the processes.
func (pm *Manager) StartAllProcesses() {
	procs := pm.sendAll(processRequest{
		kind: reqStart,
	})
	<-pm.notifier.watch(nil, procs, isReady)
}

// StopProcesses starts all matching processes.
func (pm *Manager) StartProcesses(name string, labels map[string]string) error {
	procs := pm.sendAll(processRequest{
		kind: reqStart,
	})
	if len(procs) == 0 {
		return ErrNotFound
	}
	<-pm.notifier.watch(nil, procs, isReady)
	return nil
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
	procs := pm.sendAll(processRequest{
		kind: reqStop,
	})
	if len(procs) == 0 {
		return ErrNotFound
	}
	<-pm.notifier.watch(nil, procs, isStopped)
	return nil
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

// match reports whether the p matches the given name and labels.
func (pm *Manager) match(p *process, name string, labels map[string]string) bool {
	cfg := pm.config.Programs[name]
	if name != "" && cfg.Name != name {
		return false
	}
	for k, v := range labels {
		pv, ok := cfg.Labels[k]
		if !ok || v != pv {
			return false
		}
	}
	return true
}
