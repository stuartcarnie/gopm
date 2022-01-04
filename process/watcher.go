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
	if w.state[p] != state {
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

// watch returns a channel that's closed when when the value of
// f(p[0].state) && f(p[1].state) && ... && f[p[len(procs)-1].state]
// becomes true.
//
// If the stateWatcher is closed, the channel will never
// be closed.
//
// If cancel is closed, the associated goroutine will eventually
// be shut down.
func (w *stateNotifier) watch(cancel <-chan struct{}, procs []*process, f func(State) bool) <-chan struct{} {
	c := make(chan struct{})
	go func() {
		w.mu.RLock()
		defer w.mu.RUnlock()
		for {
			if w.closed {
				return
			}
			if w.allStatesSatisfy(procs, f) {
				close(c)
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

func isReady(s State) bool {
	return s == Running || s == Exited
}

func isReadyOrFailed(s State) bool {
	return s == Running || s == Exited || s == Fatal
}

func isStopped(s State) bool {
	return s == Stopped
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
