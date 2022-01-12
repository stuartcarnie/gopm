package process

import (
	"sync"
)

// stateNotifier allows processes to notify other processes about their state changes.
type stateNotifier struct {
	changed sync.Cond
	mu      sync.RWMutex
	closed  bool
	state   map[*process]State
}

func newStateNotifier() *stateNotifier {
	w := &stateNotifier{
		state: make(map[*process]State),
	}
	w.changed.L = w.mu.RLocker()
	return w
}

// close closes w and causes all current watchers
// to return false immediately.
func (w *stateNotifier) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	w.changed.Broadcast()
}

// setState records the given state for a process. It does not block,
// but will notify any waiting watchers of the change when it can.
func (w *stateNotifier) setState(p *process, state State) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if current, ok := w.state[p]; !ok || current != state {
		w.state[p] = state
		w.changed.Broadcast()
	}
}

// remove removes the given process from the notifier.
func (w *stateNotifier) remove(p *process) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.state, p)
}

// waitFor waits for all the given processes to reach a steady state, then sends
// a slice of all the processes that haven't reached the desired state on the
// returned channel.
//
// If cancel is closed, the associated goroutine will eventually
// be shut down: no value will be sent on the returned channel.
func (w *stateNotifier) waitFor(cancel <-chan struct{}, procs []*process, desired func(State) bool) <-chan []*process {
	c := make(chan []*process, 1)
	go func() {
		w.mu.RLock()
		defer w.mu.RUnlock()
		for {
			if w.closed || w.allStatesSatisfy(procs, isSteady) {
				c <- w.unsatisfied(procs, desired)
				return
			}
			select {
			case <-cancel:
				return
			default:
			}
			w.changed.Wait()
		}
	}()
	return c
}

func (w *stateNotifier) unsatisfied(procs []*process, f func(State) bool) []*process {
	var unsatisfied []*process
	for _, p := range procs {
		state, ok := w.state[p]
		if !ok {
			// The process has gone away so we consider the state satisfied.
			continue
		}
		if !f(state) {
			unsatisfied = append(unsatisfied, p)
		}
	}
	return unsatisfied
}

// watchAll watches for changes to the current state and calls f
// every time it does. The value passed to f holds the current state of all programs
// keyed by program name.
//
// It returns a channel that is closed if w is shut down. If that happens,
// f won't be called again.
//
// f is always called immediately with the current state before waiting.
//
// If cancel is closed, the associated goroutine will eventually
// be shut down.
func (w *stateNotifier) watchAll(cancel <-chan struct{}, f func(map[string]State)) <-chan struct{} {
	closed := make(chan struct{})
	go func() {
		w.mu.RLock()
		defer w.mu.RUnlock()
		for {
			if w.closed {
				close(closed)
				return
			}
			select {
			case <-cancel:
				return
			default:
			}
			m := make(map[string]State)
			for p, state := range w.state {
				m[p.name] = state
			}
			w.mu.RUnlock()
			f(m)
			w.mu.RLock()
			w.changed.Wait()
		}
	}()
	return closed
}

func isReady(s State) bool {
	return s == Running || s == Exited
}

func isReadyOrFailed(s State) bool {
	return s == Running || s == Exited || s == Fatal
}

func isSteady(s State) bool {
	return isReadyOrFailed(s) || s == Stopped || s == Paused
}

func isStoppedOrPaused(s State) bool {
	return s == Stopped || s == Paused
}

func (w *stateNotifier) allStatesSatisfy(procs []*process, f func(State) bool) bool {
	for _, p := range procs {
		state, ok := w.state[p]
		if !ok {
			// The process has gone away so there's no point in
			// waiting for it any more.
			continue
		}
		if !f(state) {
			return false
		}
	}
	return true
}
