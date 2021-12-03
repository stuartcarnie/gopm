package process

import (
	"sync"

	"github.com/stuartcarnie/gopm/config"
	"go.uber.org/zap"
)

// Manager manage all the process in the supervisor
type Manager struct {
	procs map[string]*Process
	lock  sync.Mutex
}

// NewManager create a new Manager object
func NewManager() *Manager {
	return &Manager{
		procs: make(map[string]*Process),
	}
}

// CreateOrUpdateProcess creates a new process and adds it to the manager or updates an existing process.
func (pm *Manager) CreateOrUpdateProcess(supervisorID string, after *config.Process) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	proc, ok := pm.procs[after.Name]
	if !ok {
		proc = NewProcess(supervisorID, after)
		pm.procs[after.Name] = proc
		zap.L().Info("Created process", zap.String("name", after.Name))
	} else {
		proc.UpdateConfig(after)
		zap.L().Info("Updated process", zap.String("name", after.Name))
	}

	return proc
}

// RemoveProcess remove the process from the manager and stops it, if it was running.
// Returns the removed process.
func (pm *Manager) RemoveProcess(name string) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	proc := pm.procs[name]
	if proc != nil {
		delete(pm.procs, name)
		proc.Destroy(true)
		zap.L().Info("Removed process", zap.String("name", name))
	}

	return proc
}

// StartAutoStartPrograms start all the program if its autostart is true
func (pm *Manager) StartAutoStartPrograms() {
	pm.ForEachProcess(func(proc *Process) {
		if proc.config.AutoStart {
			proc.Start(false)
		}
	})
}

// Add add the process to this process manager
func (pm *Manager) Add(name string, proc *Process) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.procs[name] = proc
}

// Find find process by program name return process if found or nil if not found
func (pm *Manager) Find(name string) *Process {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	return pm.procs[name]
}

// FindMatchWithLabels matches Processes by name and labels.
func (pm *Manager) FindMatchWithLabels(name string, labels map[string]string) []*Process {
	if name != "" {
		p := pm.Find(name)
		// Specifying a name and some labels is probably not very useful
		// but support it anyway.
		if p == nil || !p.MatchLabels(labels) {
			return nil
		}
		return []*Process{p}
	}
	var procs []*Process
	pm.ForEachProcess(func(p *Process) {
		if p.MatchLabels(labels) {
			procs = append(procs, p)
		}
	})
	return procs
}

// Clear clear all the processes
func (pm *Manager) Clear() {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.procs = make(map[string]*Process)
}

// ForEachProcess process each process in sync mode
func (pm *Manager) ForEachProcess(procFunc func(p *Process)) {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	procs := pm.getAllProcess()
	for _, proc := range procs {
		procFunc(proc)
	}
}

// AsyncForEachProcess handle each process in async mode
// Args:
// - procFunc, the function to handle the process
// - done, signal the process is completed
// Returns: number of total processes
func (pm *Manager) AsyncForEachProcess(procFunc func(p *Process), done chan *Process) int {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	procs := pm.getAllProcess()

	for _, proc := range procs {
		go forOneProcess(proc, procFunc, done)
	}
	return len(procs)
}

func forOneProcess(proc *Process, action func(p *Process), done chan *Process) {
	action(proc)
	done <- proc
}

func (pm *Manager) getAllProcess() []*Process {
	tmpProcs := make([]*Process, 0)
	for _, proc := range pm.procs {
		tmpProcs = append(tmpProcs, proc)
	}
	return sortProcess(tmpProcs)
}

// StopAllProcesses stop all the processes managed by this manager
func (pm *Manager) StopAllProcesses() {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	processes := pm.getAllProcess()
	var wg sync.WaitGroup
	wg.Add(len(processes))
	for _, p := range processes {
		go func(proc *Process) {
			defer wg.Done()
			proc.Stop(true)
		}(p)
	}

	wg.Wait()
}

func sortProcess(procs []*Process) []*Process {
	progConfigs := make([]*config.Process, 0)
	for _, proc := range procs {
		progConfigs = append(progConfigs, proc.config)
	}

	result := make([]*Process, 0)
	p := newProcessSorter()
	for _, program := range p.sort(progConfigs) {
		for _, proc := range procs {
			if proc.config == program {
				result = append(result, proc)
			}
		}
	}

	return result
}
