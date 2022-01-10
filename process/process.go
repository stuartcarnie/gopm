package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"

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

//go:generate go run golang.org/x/tools/cmd/stringer@v0.1.8 -type processRequestKind

type processRequestKind int

const (
	reqInvalid processRequestKind = iota
	reqUpdate
	reqStart
	reqStop
	reqRestart
	reqInfo
	reqLogger
	reqSignal
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

	// reqSignal
	signal config.Signal
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

	// cronTimer expires when the next cron event happens.
	// It's non-nil only if the process has a cron entry.
	cronTimer *time.Timer

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

	// zlog is an annotated logger for the process.
	// TODO maybe this should also log to logger.
	zlog *zap.Logger
}

func (p *process) run() {
	p.zlog = zap.L().With(zap.String("prog", p.name))
	p.logger = logger.New(logger.Params{
		LogFile:     p.config.LogFile,
		MaxFileSize: p.config.LogFileMaxBytes,
		MaxBacklog:  p.config.LogFileMaxBacklogBytes,
		Backups:     p.config.LogFileBackups,
		Prefix:      p.name + ": ",
	})
	defer p.logger.Close()
	timer := time.NewTimer(time.Minute)
	timer.Stop()
	defer p.stopDepsWatch()
	p.startCron()

	// Loop waiting for events, and letting p.state largely dictate what we
	// do when things happen. Note that this goroutine should never block
	// anywhere other than in the select statement below - it should always
	// be ready to accept commands on p.req.
	for {
		p.zlog.Debug("loop", zap.Stringer("state", p.state))
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
				// It's time to start the command.
				p.startCount++
				p.totalStartCount++
				p.startTime = time.Now()
				p.stopTime = time.Time{}
				if p.config.Command == "" {
					// The empty command always succeeds immediately.
					p.state = Exited
					break
				}
				if err := p.startCommand(); err != nil {
					// It's an error that won't be fixed by retrying.
					p.zlog.Info("cannot start command", zap.Error(err))
					p.state = Fatal
					break
				}
				p.zlog.Info("start")

				// Wake up when the command has been running long enough
				// to mark it as such (or only when the command exits if it's
				// a one-shot command).
				p.startTime = time.Now()
				p.stopTime = time.Time{}
				if !p.config.Oneshot {
					timer.Reset(p.config.StartSeconds.D)
				}
			}
			if p.cmd != nil && !p.config.Oneshot && time.Since(p.startTime) >= p.config.StartSeconds.D {
				// The command has been running for long enough to go
				// into the Running state.
				p.state = Running
				p.startCount = 0
				timer.Stop()
			}
		case Stopping:
			// When we're stopping, p.stopSignals keeps track of the next signal
			// in the sequence of signals to send. It's moved on one element for
			// every stop attempt.
			if time.Since(p.killTime) < p.config.StopWaitSeconds.D {
				// We're waiting until we can try to kill with the next signal
				break
			}
			if len(p.stopSignals) == 0 {
				// We've already done all we can to kill it, so just throw it away.
				p.cmd = nil
				p.cmdWait = nil
				p.zlog.Error("killed process but it failed to exit")
				p.state = Stopped
				break
			}
			if err := p.signal(p.stopSignals[0]); err != nil {
				p.zlog.Info("failed to send stop signal", zap.Error(err))
			}
			p.stopSignals = p.stopSignals[1:]
			p.killTime = time.Now()
			timer.Reset(p.config.StopWaitSeconds.D)
		case Stopped:
		case Running:
		case Exited:
		case Fatal:
		}

		// Notify everyone else of our current state.
		p.notifier.setState(p, p.state)
		if p.state != Starting && p.state != Backoff {
			// We don't need to watch for dependencies unless we're trying
			// to start a command.
			p.stopDepsWatch()
		}
		var cronTimerC <-chan time.Time
		if p.cronTimer != nil {
			cronTimerC = p.cronTimer.C
		}
		p.zlog.Debug("waiting for event", zap.Stringer("state", p.state))
		select {
		case req, ok := <-p.req:
			if !ok {
				// The request channel has closed to signal this process has been
				// removed.
				if p.cmd != nil {
					panic("process torn down without killing process!")
				}
				return
			}
			// We've got a request from outside.
			p.zlog.Info("handle request", zap.Stringer("kind", req.kind))
			p.handleRequest(req)

		case exit := <-p.cmdWait:
			// The command that we're running has exited.

			p.zlog.Info("exited", zap.Error(exit))
			p.cmd = nil
			p.exitStatus = exit
			p.startTime = time.Time{}
			p.killTime = time.Time{}
			p.stopTime = time.Now()
			if p.state == Stopping {
				// TODO Should we log any unexpected exit status here?
				if p.needsRestart() {
					p.state = Starting
				} else {
					p.state = Stopped
				}
				break
			}
			okExit := p.isExpectedExit(exit)
			// With the default value (nil) of AutoRestart, we'll restart
			// only if the exit status isn't an expected one.
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
				p.startCount = 0
				break
			}
			// Wake up when we can try again.
			p.state = Backoff
			timer.Reset(p.config.RestartPause.D)

		case <-p.depsWatch:
			// Our dependencies have become ready.
			p.zlog.Info("dependencies are ready")
			p.depsRunning = true

		case <-timer.C:
			// A previously configured timer event has fired.
		case <-cronTimerC:
			// A cron event has triggered; it's equivalent to a start request.
			p.handleRequest(processRequest{
				kind: reqStart,
			})
			p.cronTimer.Reset(p.nextCronWait())
		}
	}
}

// setStopSignals sets p.stopSignals to the list of signals that
// we'll use to try to stop the process. It also makes sure that
// there's always a KILL signal as the last resort.
func (p *process) setStopSignals() {
	sigs := p.config.StopSignals
	if len(sigs) > 0 && sigs[len(sigs)-1].S == os.Kill {
		p.stopSignals = sigs
		return
	}
	sigs = append([]config.Signal{}, sigs...)
	sigs = append(sigs, killSig)
	p.stopSignals = sigs
}

// signal sends a signal to the running command.
func (p *process) signal(sig config.Signal) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	p.zlog.Info("killing process", zap.String("signal", sig.String()))
	return signals.Kill(p.cmd.Process, sig.S, true)
}

// startDepsWatch starts a watcher to notify this process when
// its dependencies are all running.
func (p *process) startDepsWatch() {
	p.stopDepsWatch()
	p.watchStopper = make(chan struct{})
	p.depsWatch = p.notifier.watch(p.watchStopper, p.dependsOn, isReady)
	p.depsRunning = false // Assume they're not running until proven otherwise.
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
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Dir = p.config.Directory
	cmd.Stdout = p.logger
	cmd.Stderr = p.logger
	setProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start command: %v", err)
	}

	// Start a goroutine notify us when the process has exited.
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
			// No need to do anything - we're already started or are trying to start.
		default:
			// Ignore any current exited, failed or backoff status.
			p.state = Starting
		}
	case reqStop, reqRestart:
		if p.cmd == nil || p.state == Backoff {
			p.state = Stopped
		} else {
			p.state = Stopping
			p.setStopSignals()
		}
		p.setNeedsRestart(req.kind == reqRestart)
	case reqLogger:
		req.loggerReply <- p.logger
	case reqInfo:
		req.infoReply <- p.info()
	case reqSignal:
		p.signal(req.signal) // Note: ignore error.
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
		Name:        p.name,
		Description: p.config.Description,
		Start:       p.startTime,
		Stop:        p.stopTime,
		State:       p.state,
		ExitStatus:  exitCode(p.exitStatus),
		Logfile:     p.config.LogFile,
		Pid:         pid,
	}
}

// The program's configuration has changed.
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
	p.startCron()
}

func (p *process) startCron() {
	if p.cronTimer != nil {
		p.cronTimer.Stop()
		p.cronTimer = nil
	}
	if p.config.Cron != nil {
		p.cronTimer = time.NewTimer(p.nextCronWait())
	}
}

func (p *process) nextCronWait() time.Duration {
	now := time.Now()
	t := p.config.Cron.Schedule.Next(now)
	return t.Sub(now)
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

var killSig = func() config.Signal {
	sig, err := config.ParseSignal("KILL")
	if err != nil {
		panic(err)
	}
	return sig
}()
