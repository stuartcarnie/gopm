package process

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/stuartcarnie/gopm/config"
	"go.uber.org/zap"
)

// ErrNotFound is returned when a multi-process operation is
// executed but no processes were found to operate on.
var ErrNotFound = errors.New("no matching process found")

// Manager manage all the process in the supervisor
type Manager struct {
	// Note: this should really be guarded by lock because
	// Update can be called concurrently, but that's an old
	// issue and we're going to rip all this out anyway.
	config *config.Config

	procs map[string]*process
	lock  sync.Mutex
}

// NewManager returns a new Manager holding no processes.
func NewManager() *Manager {
	return &Manager{
		procs: make(map[string]*process),
	}
}

// Update updates the configuration, creating, removing and restarting
// processes as appropriate.
func (pm *Manager) Update(newConfig *config.Config) {
	if pm.config != nil {
		for name := range pm.config.Programs {
			if newConfig.Programs[name] == nil {
				pm.removeProcess(name)
			}
		}
	}
	for _, p := range newConfig.Programs {
		// TODO remove the redundant supervisorID argument.
		pm.createOrUpdateProcess("supervisor", p)
	}
	pm.config = newConfig
	pm.startAutoStartPrograms()
}

// createOrUpdateProcess creates a new process and adds it to the manager or updates an existing process.
func (pm *Manager) createOrUpdateProcess(supervisorID string, after *config.Program) *process {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	proc, ok := pm.procs[after.Name]
	if !ok {
		proc = newProcess(supervisorID, after)
		pm.procs[after.Name] = proc
		zap.L().Info("Created process", zap.String("name", after.Name))
	} else {
		proc.updateConfig(after)
		zap.L().Info("Updated process", zap.String("name", after.Name))
	}

	return proc
}

// removeProcess removes the process from the manager and stops it, if it was running.
// Returns the removed process.
func (pm *Manager) removeProcess(name string) *process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	proc := pm.procs[name]
	if proc != nil {
		delete(pm.procs, name)
		proc.destroy(true)
		zap.L().Info("Removed process", zap.String("name", name))
	}

	return proc
}

// startAutoStartPrograms start all the programs when their autostart flag is true.
// TODO this should be done by the Process itself.
func (pm *Manager) startAutoStartPrograms() {
	pm.forEachProcess(func(proc *process) {
		if proc.config.AutoStart {
			proc.start(false)
		}
	})
}

func (pm *Manager) forEachMatch(name string, labels map[string]string, f func(p *process)) error {
	procs := pm.findMatchWithLabels(name, labels)
	if len(procs) == 0 {
		return ErrNotFound
	}
	for _, p := range procs {
		f(p)
	}
	return nil
}

// find returns the process with the given name, or nil if there are none.
func (pm *Manager) find(name string) *process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	return pm.procs[name]
}

// findMatchWithLabels returns processes matched by name and labels.
// If name is empty, all processes matching any of the labels will be returned.
func (pm *Manager) findMatchWithLabels(name string, labels map[string]string) []*process {
	if name != "" {
		p := pm.find(name)
		// Specifying a name and some labels is probably not very useful
		// but support it anyway.
		if p == nil || !p.matchLabels(labels) {
			return nil
		}
		return []*process{p}
	}
	var procs []*process
	pm.forEachProcess(func(p *process) {
		if p.matchLabels(labels) {
			procs = append(procs, p)
		}
	})
	return procs
}

// forEachProcess calls f for each process in the manager with
// the mutex obtained.
func (pm *Manager) forEachProcess(f func(p *process)) {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	for _, proc := range pm.allProcesses() {
		f(proc)
	}
}

func (pm *Manager) allProcesses() []*process {
	procs := make([]*process, 0, len(pm.procs))
	for _, proc := range pm.procs {
		procs = append(procs, proc)
	}
	return procs
}

// StopProcesses starts all matching processes.
func (pm *Manager) StartProcesses(name string, labels map[string]string) error {
	return pm.forEachMatch(name, labels, func(p *process) {
		p.start(true)
	})
}

// StopProcesses stops all matching processes.
func (pm *Manager) StopProcesses(name string, labels map[string]string) error {
	return pm.forEachMatch(name, labels, func(p *process) {
		p.stop(true)
	})
}

// RestartProcesses restarts all matching processes.
func (pm *Manager) RestartProcesses(name string, labels map[string]string) error {
	return pm.forEachMatch(name, labels, func(p *process) {
		p.stop(true)
		p.start(true)
	})
}

// SignalProcesses sends the given signal to all matching processes.
func (pm *Manager) SignalProcesses(name string, labels map[string]string, sig os.Signal) error {
	return pm.forEachMatch(name, labels, func(p *process) {
		p.signal(sig)
	})
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
	proc := pm.find(p.Name)
	if proc == nil {
		return ErrNotFound
	}
	if p.BacklogLines > 0 {
		_, data := proc.outputBacklog.Bytes()
		data = lastNLines(data, int(p.BacklogLines))
		if _, err := p.Writer.Write(data); err != nil {
			return err
		}
		if !p.Follow {
			return nil
		}
		// We can miss some data here because there can be a write
		// between the Bytes call above and adding the new logger.
		// TODO fix it so that we can't miss data.
	}

	plog := nopCloser{p.Writer}
	proc.outputLog.AddLogger(plog)
	<-ctx.Done()
	proc.outputLog.RemoveLogger(plog)
	return nil
}

// StopAllProcesses stops all the processes managed by this manager
func (pm *Manager) StopAllProcesses() {
	var wg sync.WaitGroup
	pm.forEachProcess(func(p *process) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.stop(true)
		}()
	})
	wg.Wait()
}

// AllProcessInfo returns information on all the processes.
func (pm *Manager) AllProcessInfo() []*ProcessInfo {
	var infos []*ProcessInfo
	pm.forEachProcess(func(p *process) {
		infos = append(infos, p.info())
	})
	return infos
}

// StartAllProcesses starts all the processes.
func (pm *Manager) StartAllProcesses() {
	var wg sync.WaitGroup
	pm.forEachProcess(func(p *process) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.start(true)
		}()
	})
	wg.Wait()
}

type nopCloser struct {
	io.Writer
}

func (c nopCloser) Close() error {
	return nil
}
