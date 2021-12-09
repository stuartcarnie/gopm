package process

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/logger"
	"github.com/stuartcarnie/gopm/signals"
)

// State represents the state of a process
type State int

const (
	Stopped State = iota
	Starting
	Running
	Backoff
	Stopping
	Exited
	Fatal
)

// String returns p as a human readable string
func (p State) String() string {
	switch p {
	case Stopped:
		return "Stopped"
	case Starting:
		return "Starting"
	case Running:
		return "Running"
	case Backoff:
		return "Backoff"
	case Stopping:
		return "Stopping"
	case Exited:
		return "Exited"
	case Fatal:
		return "Fatal"
	default:
		return "Unknown"
	}
}

//go:generate stringer -type processRequestKind

type processRequestKind int

const (
	reqInvalid processRequestKind = iota
	reqUpdate
	reqStart
	reqStop
	reqRestart
	reqInfo
	reqLogger
)

type processRequest struct {
	kind processRequestKind

	// reqUpdate
	newConfig *config.Program
	newDeps   []*process

	// reqInfo
	infoReply chan<- *ProcessInfo

	// reqLogger
	loggerReply chan<- *logger.Logger
}

type process struct {
	// name holds the process's name.
	name string

	// notifier is used to inform other processes of any state changes.
	notifier *stateNotifier

	// req holds the channel for making requests to
	// the process's manager goroutine.
	req chan processRequest

	// The following fields are managed by the process's manager
	// goroutine:

	// config holds the program's configuration.
	// It's nil when the process is initially created.
	config *config.Program

	// state holds the current state of the process.
	state State

	// dependsOn holds the process's dependencies.
	dependsOn []*process

	// cmd holds the currently running command if any.
	cmd *exec.Cmd

	// cmdWait receives a value when the above command finishes.
	// It's recreated every time a command is started.
	cmdWait chan error

	// exitStatus holds the error returned by the last cmd.Wait call.
	exitStatus error

	// startTime holds the time that the command was started, or the
	// zero time if there is no command running.
	startTime time.Time

	// stopTime holds the time that the last command exited,
	// or the zero time if a command is currently running.
	stopTime time.Time

	// killTime holds the time that the most recent kill attempt
	// was made when trying to stop the process.
	killTime time.Time

	// stopSignals holds the remaining signals we're planning to
	// send if the process refuses to exit.
	stopSignals []config.Signal

	// startCount holds the number of start attempts since the process
	// was moved into Starting state.
	startCount int

	// depsRunning holds whether all the dependencies are considered
	// to be currently running.
	depsRunning bool

	// totalStartCount holds the number of times the command
	// has been started.
	totalStartCount int

	// wantStartCount holds the number of times we want the
	// command to have been started.
	wantStartCount int

	// depsWatch is a channel returned by p.notifier.watch
	// and is closed when all our dependencies
	// are ready.
	depsWatch <-chan struct{}

	// watchStopper is used to tear down the above watcher.
	watchStopper chan struct{}

	// logger is used for logging process output.
	logger *logger.Logger
}

func (p *process) run() {
	p.logger = logger.New(
		p.config.LogFile,
		p.config.LogFileMaxBytes,
		p.config.LogFileMaxBacklogBytes,
		p.config.LogFileBackups,
	)
	timer := time.NewTimer(time.Minute)
	timer.Stop()
	defer p.stopDepsWatch()
	for {
		switch p.state {
		case Starting, Backoff:
			if p.depsWatch == nil {
				// We don't know whether all our dependencies have started, so start a watcher
				// to find out.
				p.startDepsWatch()
				break
			}
			if !p.depsRunning {
				break
			}
			p.stopDepsWatch()
			if p.cmd == nil && time.Since(p.stopTime) >= p.config.RestartPause.D {
				p.startCount++
				p.totalStartCount++
				if err := p.startCommand(); err != nil {
					p.state = Fatal
				}

				p.startTime = time.Now()
				timer.Reset(p.config.StartSeconds.D)
				p.stopTime = time.Time{}
			}
			if time.Since(p.startTime) >= p.config.StartSeconds.D {
				p.state = Running
				p.startCount = 0
				timer.Stop()
			}
		case Stopping:
			if time.Since(p.killTime) < p.config.StopWaitSeconds.D {
				// We're waiting until we can try to kill with the next signal
				break
			}
			if len(p.stopSignals) == 0 {
				// We've already done all we can to kill it, so just throw it away.
				p.cmd = nil
				p.cmdWait = nil
				// TODO log this
				p.state = Stopped
				break
			}
			if err := p.signal(p.stopSignals[0]); err != nil {
				log.Printf("cannot signal process: %v", err)
			}
			p.stopSignals = p.stopSignals[1:]
			p.killTime = time.Now()
			timer.Reset(p.config.StopWaitSeconds.D)
		case Stopped:
		case Running:
		case Exited:
		case Fatal:
		}
		p.notifier.setState(p, p.state)
		if p.state != Starting && p.state != Backoff {
			p.stopDepsWatch()
		}
		select {
		case req, ok := <-p.req:
			if !ok {
				if p.cmd != nil {
					panic("process torn down without killing process!")
				}
				return
			}
			p.handleRequest(req)
		case exit := <-p.cmdWait:
			p.cmd = nil
			p.exitStatus = exit
			p.startTime = time.Time{}
			p.stopTime = time.Now()
			if p.state == Stopping {
				// TODO Should we log any unexpected exit status here?
				if p.needsRestart() {
					p.state = Starting
					p.setStopSignals()
				} else {
					p.state = Stopped
				}
				break
			}
			okExit := p.isExpectedExit(exit)
			restart := !okExit
			if p.config.AutoRestart != nil {
				restart = *p.config.AutoRestart
			}
			if !restart || p.startCount >= p.config.StartRetries {
				if okExit {
					p.state = Exited
				} else {
					p.state = Fatal
				}
				break
			}
			p.state = Backoff
			timer.Reset(p.config.RestartPause.D)
		case <-p.depsWatch:
			p.depsRunning = true
		case <-timer.C:
		}
	}
}

func (p *process) setStopSignals() {
	sigs := p.config.StopSignals
	if len(sigs) > 0 && sigs[len(sigs)-1].S == os.Kill {
		p.stopSignals = sigs
		return
	}
	sigs = append([]config.Signal{}, sigs...)
	sigs = append(sigs, config.Signal{
		S:      os.Kill,
		String: "KILL",
	})
	p.stopSignals = sigs
}

func (p *process) signal(sig config.Signal) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	return signals.Kill(p.cmd.Process, sig.S, true)
}

func (p *process) startDepsWatch() {
	if p.watchStopper != nil {
		close(p.watchStopper)
	}
	p.watchStopper = make(chan struct{})
	p.depsWatch = p.notifier.watch(p.watchStopper, p.dependsOn, isReady)
	p.depsRunning = false // Assume they're not running until proven otherwise.
}

func (p *process) isExpectedExit(err error) bool {
	if err == nil {
		// TODO Should we really treat a zero exit code as "expected"
		// if it's not in ExitCodes?
		return true
	}
	ecode := exitCode(err)
	for _, code := range p.config.ExitCodes {
		if code == ecode {
			return true
		}
	}
	return false
}

func (p *process) stopDepsWatch() {
	p.depsRunning = false
	if p.watchStopper == nil {
		return
	}
	close(p.watchStopper)
	p.watchStopper = nil
	p.depsWatch = nil
	p.depsRunning = false
}

func (p *process) setNeedsRestart(need bool) {
	if need {
		p.wantStartCount = p.totalStartCount + 1
	} else {
		p.wantStartCount = p.totalStartCount
	}
}

func (p *process) needsRestart() bool {
	return p.totalStartCount < p.wantStartCount
}

func (p *process) startCommand() error {
	if p.cmd != nil {
		panic("startCommand with command already running")
	}
	cmd := exec.Command(p.config.Shell, "-c", p.config.Command)
	if info, err := os.Stat(p.config.Directory); err != nil {
		return fmt.Errorf("invalid directory directory for process %q: %v", p.name, err)
	} else if !info.IsDir() {
		return fmt.Errorf("invalid directory directory for process %q: %q is not a directory", p.name, p.config.Directory)
	}

	// TODO set user ID
	for k, v := range p.config.Environment {
		cmd.Env = append(p.cmd.Env, k+"="+v)
	}
	cmd.Dir = p.config.Directory
	cmd.Stdout = p.logger
	cmd.Stderr = p.logger
	setProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start command: %v", err)
	}
	cmdWait := make(chan error, 1)
	go func() {
		cmdWait <- cmd.Wait()
	}()
	p.cmd = cmd
	p.cmdWait = cmdWait
	return nil
}

func (p *process) handleRequest(req processRequest) {
	switch req.kind {
	case reqUpdate:
		p.handleUpdate(req.newConfig, req.newDeps)
	case reqStart:
		switch p.state {
		case Running, Starting:
			// No need to do anything - we're already started or trying to start.
		default:
			// Ignore any current exited, failed or backoff status.
			p.state = Starting
		}
	case reqStop, reqRestart:
		if p.cmd == nil {
			p.state = Stopped
		} else if p.state != Backoff {
			p.state = Stopping
			p.setStopSignals()
		}
		p.setNeedsRestart(req.kind == reqRestart)
	case reqLogger:
		req.loggerReply <- p.logger
	case reqInfo:
		req.infoReply <- p.info()
	default:
		panic(fmt.Errorf("unknown process request received: %v", req.kind))
	}
}

func (p *process) info() *ProcessInfo {
	pid := 0
	if p.cmd != nil {
		pid = p.cmd.Process.Pid
	}
	return &ProcessInfo{
		Name:       p.name,
		Start:      p.startTime,
		Stop:       p.stopTime,
		ExitStatus: exitCode(p.exitStatus),
		Logfile:    p.config.LogFile,
		Pid:        pid,
	}
}

func (p *process) handleUpdate(newConfig *config.Program, deps []*process) {
	if p.cmd != nil {
		// We should never be sent an update when we're not stopped.
		panic(fmt.Errorf("configuration update sent while process is in state %v", p.state))
	}
	p.state = Stopped // Note: the updater will start us if we were started before.
	p.config = newConfig
	p.dependsOn = deps
	p.exitStatus = nil
	p.startCount = 0
	p.depsRunning = false
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return -1
	}
	return exitError.ExitCode()
}
