package process

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/stuartcarnie/gopm/config"
	"go.uber.org/zap"
)

// UpdatePrograms updates the programs run by the manager.
func (pm *Manager) Update(ctx context.Context, config *config.Config) (_err error) {
	zap.L().Debug("Manager.Update", zap.Int("program-count", len(config.Programs)))
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
//
// It also monitors the current state of all processes and notifies processes
// when their dependencies have changed readiness.
func (pm *Manager) updateProc() {
	var (
		cancelUpdate func()
		currReq      *updateConfigReq
		updateDone   chan error
		currentState = make(map[string]State)
		stateChanged = make(chan map[string]State)
	)
	pm.notifier.watchAll(nil, func(states map[string]State) {
		stateChanged <- states
	})

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
			donec := make(chan error, 1)
			updateDone = donec
			go func() {
				donec <- pm.updateConfig(ctx, req.config)
			}()
		case err := <-updateDone:
			// Update complete, so reply to the originator and reset everything.
			currReq.reply <- err
			cancelUpdate()
			cancelUpdate = nil
			currReq = nil
			updateDone = nil
		case currentState = <-stateChanged:
		}
		if currReq == nil {
			// No current update in progress, so update everyone's readiness status.
			pm.updateDepsReady(currentState)
		}
	}
}

func (pm *Manager) updateDepsReady(currentState map[string]State) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	// TODO resolve config with existing deps
	// Tell all processes whether their deps are ready,
	// causing paused processes to resume if appropriate.
	for p, depsReady := range pm.getDepsReady(currentState) {
		p.req <- processRequest{
			kind:      reqDepsReady,
			depsReady: depsReady,
		}
	}
}

// getDepsReady returns a map holding whether each process in pm is ready with
// respect to its dependencies and the given state.
func (pm *Manager) getDepsReady(state map[string]State) map[*process]bool {
	ready := make(map[*process]bool)
	for _, p := range pm.processes {
		pm.markReady(p, ready, state)
	}
	return ready
}

// markReady marks in ready whether p's dependencies are ready
// and returns whether p itself is ready.
func (pm *Manager) markReady(p *process, ready map[*process]bool, state map[string]State) bool {
	pDepsReady, ok := ready[p]
	if ok {
		return pDepsReady && isReady(state[p.name])
	}
	pDepsReady = true
	for dep := range pm.config.Programs[p.name].DependsOn {
		p := pm.processes[dep]
		if p == nil {
			pDepsReady = false
			continue
		}
		pDepsReady = pm.markReady(p, ready, state) && pDepsReady
	}
	ready[p] = pDepsReady
	return pDepsReady && isReady(state[p.name])
}

func (pm *Manager) updateConfig(ctx context.Context, newConfig *config.Config) error {
	// TODO find current state of all processes so we can restart them
	// if they were already started.

	pm.mu.RLock()
	_, err := pm.pauseChangedProcesses(ctx, newConfig)
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
	changed, err := pm.pauseChangedProcesses(ctx, newConfig)
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
	//
	// Note that each process will independently wait
	// for its own dependencies to become ready before
	// starting.
	for _, newp := range newConfig.Programs {
		name := newp.Name
		if !changed[name] && pm.processes[name] != nil {
			continue
		}
		if oldp := pm.processes[name]; oldp != nil {
			// Update an existing process, which also implicitly
			// marks its dependencies as not ready.
			oldp.req <- processRequest{
				kind:      reqUpdate,
				newConfig: newp,
			}
			oldp.req <- processRequest{
				kind: reqResume,
			}
		} else {
			// It's a new process, so create it.
			pm.processes[name] = newProcess(newp, pm.notifier)
		}
	}
	pm.config = newConfig
	return nil
}

// pauseChangedProcesses pauses any processes that need to be updated.
// It returns a map keyed by the names of all such processes.
func (pm *Manager) pauseChangedProcesses(ctx context.Context, newConfig *config.Config) (map[string]bool, error) {
	return pm.stopOrPauseProcesses(ctx, func(p *config.Program) bool {
		return !reflect.DeepEqual(newConfig.Programs[p.Name], pm.config.Programs[p.Name])
	}, reqPause)
}

// stopOrPauseProcesses sends the given stop request (either reqStop or reqPause)
// to process p for which shouldStop(p) returns true. It sends reqPause to all the processes that
// transitively depend on those. It stops dependees before the processes that they depend on.
// It returns a map keyed by the processes that have been stopped or paused.
//
// This method must be called with pm.mu locked (either read-only or read-write).
func (pm *Manager) stopOrPauseProcesses(ctx context.Context, shouldStop func(p *config.Program) bool, stopReq processRequestKind) (map[string]bool, error) {
	stopping := make(map[string]bool) // all stopped processes.
	deps := getDeps(pm.config, shouldStop)
	for name, kind := range deps {
		if kind != depDependedOn {
			stopping[name] = true
		}
	}

	// In reverse topological order, stop each process that's marked to be stopped.
	progCohorts := pm.config.TopoSortedPrograms
	for i := len(progCohorts) - 1; i >= 0; i-- {
		var waitFor []*process
		stateUpdated := make(chan struct{}, len(progCohorts[i]))
		for _, p := range progCohorts[i] {
			name := p.Name
			oldp := pm.processes[name]
			if oldp == nil {
				panic(fmt.Errorf("process %v not found", name))
			}
			if !stopping[name] {
				continue
			}
			kind := stopReq
			if deps[name]&depTarget == 0 {
				// Pause it because it depends on the target set of processes
				// but isn't targeted itself.
				kind = reqPause
			}
			oldp.req <- processRequest{
				kind:         kind,
				stateUpdated: stateUpdated,
			}
			waitFor = append(waitFor, oldp)
		}
		for range waitFor {
			<-stateUpdated
		}
		select {
		case notStopped := <-pm.notifier.waitFor(ctx.Done(), waitFor, isStoppedOrPaused):
			if len(notStopped) > 0 {
				// Someone has started a process while we're stopping things in preparation
				// for the configuration update. It's hard to know what to do in this case,
				// so just return an error for now. Although this can leave things in a partially
				// shut down state, it's probably not too bad from the client's perspective,
				// as repeating their command will still get them to the same place.
				//
				// TODO this can also happen when a process starts itself as a result
				// of a cron entry. More thought required as to how we want things to
				// behave in that case.
				return nil, fmt.Errorf("some processes were started unexpectedly after asking them to stop (%v)", strings.Join(procNames(notStopped), ", "))
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return stopping, nil
}

//go:generate go run golang.org/x/tools/cmd/stringer@v0.1.8 -type depKind

type depKind int

const (
	depDependsOn depKind = 1 << iota
	depDependedOn
	depTarget
)

// getDeps returns map that holds the relationship of each program in cfg
// with any program for which isTarget returns true.
//
// Note that it's possible for a program to be all of a dependency, a dependant
// and a target.
func getDeps(cfg *config.Config, isTarget func(*config.Program) bool) map[string]depKind {
	deps := make(map[string]depKind)
	for _, p := range cfg.Programs {
		walkDeps(cfg, p, isTarget, false, deps)
	}
	return deps
}

// walkDeps is used by getDeps to walk the dependencies of p and populate deps.
// isChild holds whether p is a child of some target; it returns whether p or any child of
// is a target.
func walkDeps(cfg *config.Config, p *config.Program, isTarget func(*config.Program) bool, isChild bool, deps map[string]depKind) bool {
	if isChild {
		deps[p.Name] |= depDependedOn
	}
	pIsTarget := isTarget(p)
	if pIsTarget {
		deps[p.Name] |= depTarget
		isChild = true
	}
	isParent := false
	for name := range p.DependsOn {
		isParent = walkDeps(cfg, cfg.Programs[name], isTarget, isChild, deps) || isParent
	}
	if isParent {
		deps[p.Name] |= depDependsOn
		return true
	}
	return pIsTarget
}
