package process

import (
	"context"
	"fmt"
	"reflect"

	"github.com/stuartcarnie/gopm/config"
	"go.uber.org/zap"
)

// UpdatePrograms updates the programs run by the manager.
func (pm *Manager) Update(ctx context.Context, config *config.Config) error {
	zap.L().Debug("Manager.Update")
	reply := make(chan error, 1)
	pm.updateConfigc <- &updateConfigReq{
		config: config,
		reply:  reply,
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("update canceled by caller; operation continues: %v", ctx.Err())
	case err := <-reply:
		return err
	}
}

type updateConfigReq struct {
	config *config.Config
	reply  chan error
}

// updateProc runs as a goroutine that's responsible for managing the updating
// of the entire configuration. It only lets one configuration change proceed
// at any one time, aborting a previous change if a new one arrives.
func (pm *Manager) updateProc() {
	var (
		cancelUpdate func()
		currReq      *updateConfigReq
		updateDone   chan struct{}
	)
	defer func() {
		if currReq != nil {
			cancelUpdate()
			currReq.reply <- fmt.Errorf("update canceled by shutdown")
		}
	}()
	for {
		select {
		case req, ok := <-pm.updateConfigc:
			if !ok {
				return
			}
			if currReq != nil {
				// There's a request in progress. Abort it.
				cancelUpdate()
				// Wait for it to actually finish.
				<-updateDone
				currReq.reply <- fmt.Errorf("aborted by another concurrent config update")
				cancelUpdate = nil
				currReq = nil
				updateDone = nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancelUpdate = cancel
			currReq = req
			donec := make(chan struct{}, 1)
			updateDone = donec
			go func() {
				defer close(donec)
				pm.updateConfig(ctx, req.config)
			}()
		case <-updateDone:
			// Update complete, so reply to the originator and reset everything.
			currReq.reply <- nil
			cancelUpdate()
			cancelUpdate = nil
			currReq = nil
			updateDone = nil
		}
	}
}

func (pm *Manager) updateConfig(ctx context.Context, newConfig *config.Config) error {
	// TODO find current state of all processes so we can restart them
	// if they were already started.

	pm.mu.RLock()

	// Find out the current state of all processes so that we can
	// restart them if needed after updating their configuration.
	reply := make(chan *ProcessInfo)
	req := processRequest{
		kind:      reqInfo,
		infoReply: reply,
	}
	for _, p := range pm.processes {
		p.req <- req
	}
	needStart := make(map[string]bool)
	for i := 0; i < len(pm.processes); i++ {
		info := <-reply
		needStart[info.Name] = info.State != Stopping && info.State != Stopped
	}

	_, err := pm.stopChangedProcesses(ctx, newConfig)
	pm.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("cannot stop processes: %v", err)
	}
	// All processes are now stopped, so we should be able to remove
	// the processes that are no longer needed.
	// However it's possible that a client is being unhelpful and
	// has just started a process that we've just stopped, so
	// stop the processes again, but under the exclusive mutex this time.
	// In all practical situations, this should finish immediately.
	//
	// Note that the exclusive mutex guarantees that no more changes
	// can be made because the process.req channel is synchronous (length 0)
	// and requests sent to a process are sent under read lock,
	// so we know that a process must have started dealing with
	// the request while the lock is held and hence no requests can arrive
	// after we acquire the lock here.
	pm.mu.Lock()
	defer pm.mu.Unlock()
	changed, err := pm.stopChangedProcesses(ctx, newConfig)
	if err != nil {
		return fmt.Errorf("cannot stop processes (second time): %v", err)
	}
	// Remove processes that are no longer needed.
	for name, oldp := range pm.processes {
		if newConfig.Programs[name] == nil {
			close(oldp.req)
			delete(pm.processes, name)
			pm.notifier.remove(oldp)
		}
	}
	// Create any new processes and update the old ones
	// (in topological order so we can present each process with
	// the correct dependencies).
	//
	// Note that each process will independently wait
	// for its own dependencies to become ready before
	// starting.
	for _, newps := range newConfig.TopoSortedPrograms {
		for _, newp := range newps {
			name := newp.Name
			if !changed[name] && pm.processes[name] != nil {
				continue
			}
			deps := make([]*process, len(newp.DependsOn))
			for i, dep := range newp.DependsOn {
				deps[i] = pm.processes[dep]
				if deps[i] == nil {
					panic("topological order issue?")
				}
			}
			oldp := pm.processes[name]
			if oldp == nil {
				// It's a new process, so create it.
				initialState := Stopped
				if newp.AutoStart {
					initialState = Starting
				}
				newp := &process{
					name:      name,
					notifier:  pm.notifier,
					req:       make(chan processRequest),
					config:    newp,
					state:     initialState,
					dependsOn: deps,
				}
				pm.notifier.setState(newp, initialState)
				pm.processes[name] = newp
				go newp.run()
				continue
			}
			// Update an existing process.
			oldp.req <- processRequest{
				kind:      reqUpdate,
				newConfig: newp,
				newDeps:   deps,
			}
			if needStart[name] {
				// The process wasn't stopped before, so start it again now.
				oldp.req <- processRequest{
					kind: reqStart,
				}
			}
		}
	}
	pm.config = newConfig
	return nil
}

// stopChangedProcesses stops any processes that need to be updated.
// It returns a map keyed by the names of all such processes.
func (pm *Manager) stopChangedProcesses(ctx context.Context, newConfig *config.Config) (map[string]bool, error) {
	return pm.stopProcesses(ctx, func(p *config.Program) bool {
		return !reflect.DeepEqual(newConfig.Programs[p.Name], pm.config.Programs[p.Name])
	})
}

// stopProcesses stops every process p for which shouldStop(p) returns true and all the processes that
// transitively depend on those. It stops dependees before the processes that they depend on.
// It returns a map keyed by all the processes that have been stopped.
//
// This method must be called with pm.mu locked (either read-only or read-write).
func (pm *Manager) stopProcesses(ctx context.Context, shouldStop func(p *config.Program) bool) (map[string]bool, error) {
	stopping := make(map[string]bool)
	for _, p := range pm.config.Programs {
		if shouldStop(p) {
			stopping[p.Name] = true
		}
	}
	// Mark every process that depends on the stopping processes
	// to be stopped too.
	for _, p := range pm.config.Programs {
		if pm.dependsOn(p, stopping) {
			stopping[p.Name] = true
		}
	}

	// In reverse topological order, stop each process that's marked to be stopped.
	progCohorts := pm.config.TopoSortedPrograms
	for i := len(progCohorts) - 1; i >= 0; i-- {
		var waitFor []*process
		for _, p := range progCohorts[i] {
			name := p.Name
			oldp := pm.processes[name]
			if oldp == nil {
				panic(fmt.Errorf("process %v not found", name))
			}
			if !stopping[name] {
				continue
			}
			oldp.req <- processRequest{
				kind: reqStop,
			}
			waitFor = append(waitFor, oldp)
		}
		select {
		case <-pm.notifier.watch(ctx.Done(), waitFor, isStopped):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return stopping, nil
}

// dependsOn reports whether the process with the given name depends on
// any of the processes with names in processNames.
func (pm *Manager) dependsOn(p *config.Program, processNames map[string]bool) bool {
	found := false
	pm.walkDependencies(p, func(p *config.Program) bool {
		if processNames[p.Name] {
			found = true
			return false
		}
		return true
	})
	return found
}

// walkDependencies calls f for all dependencies of p, which must exist within
// p.config. If f returns false, walkDependencies will return immediately.
//
// walkDependencies must be called with pm.mu held (read-only or read-write).
func (pm *Manager) walkDependencies(p *config.Program, f func(*config.Program) bool) bool {
	if !f(p) {
		return false
	}
	for _, dep := range p.DependsOn {
		if !pm.walkDependencies(pm.config.Programs[dep], f) {
			return false
		}
	}
	return true
}
