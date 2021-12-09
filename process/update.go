package process

import (
	"context"
	"fmt"
	"reflect"

	"github.com/stuartcarnie/gopm/config"
)

// UpdatePrograms updates the programs run by the manager.
func (pm *Manager) Update(ctx context.Context, config *config.Config) error {
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
		}
	}
	pm.config = newConfig
	return nil
}

// stopChangedProcesses stops any processes that need to be updated.
// It returns a map keyed by the names of all such processes.
func (pm *Manager) stopChangedProcesses(ctx context.Context, newConfig *config.Config) (map[string]bool, error) {
	changed := make(map[string]bool)
	for name, oldp := range pm.config.Programs {
		newp := newConfig.Programs[name]
		if reflect.DeepEqual(newp, oldp) {
			// TODO we could potentially be less conservative and avoid
			// tearing down the process when a new command doesn't actually
			// need to be run.
			continue
		}
		changed[name] = true
	}
	if len(changed) == 0 {
		return nil, nil
	}
	for name := range pm.processes {
		walkChanged(pm.config, name, changed)
	}
	// In reverse topological order, stop each process that's marked to be stopped.
	progCohorts := pm.config.TopoSortedPrograms
	for i := len(progCohorts) - 1; i >= 0; i-- {
		var waitFor []*process
		for _, p := range progCohorts[i] {
			name := p.Name
			if !changed[name] {
				continue
			}
			oldp := pm.processes[name]
			if oldp == nil {
				panic(fmt.Errorf("process %v not found", name))
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
	return changed, nil
}

// walkChanged sets any entries in the changed map that correspond to
// names of processes with any dependencies that have changed.
// This could be more efficient, but that almost certainly doesn't matter.
func walkChanged(cfg *config.Config, name string, changed map[string]bool) bool {
	p := cfg.Programs[name]
	pchanged := false
	for _, dep := range p.DependsOn {
		pchanged = walkChanged(cfg, dep, changed) || pchanged
	}
	if pchanged {
		changed[name] = true
	}
	return pchanged
}
