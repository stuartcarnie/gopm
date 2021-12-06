package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/logger"
	"github.com/stuartcarnie/gopm/signals"
	"go.uber.org/zap"
)

// State the state of process
type State int

const (
	Stopped State = iota
	Starting
	Running
	Backoff
	Stopping
	Exited
	Fatal

	// TODO make this configurable, or (perhaps better) just use the
	// log file directly rather than having a separate in-memory ring buffer
	// implementation.
	backlogBytes = 1024 * 1024 // 1MiB
)

var scheduler *cron.Cron = nil

func init() {
	scheduler = cron.New(cron.WithSeconds())
	scheduler.Start()
}

// String convert State to human readable string
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

type process struct {
	supervisorID string
	cmd          *exec.Cmd
	log          *zap.Logger
	startTime    time.Time
	stopTime     time.Time
	state        State
	cronID       cron.EntryID
	// true if process is starting
	inStart bool
	// true if the process is stopped by user
	stopByUser       bool
	retryTimes       *int32
	mu               sync.RWMutex
	stdin            io.WriteCloser
	outputLog        *logger.CompositeLogger
	currentOutputLog logger.Logger
	outputBacklog    *ringBuffer
	cfgMu            sync.RWMutex // protects config access
	config           *config.Program
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

func newProcess(supervisorID string, cfg *config.Program) *process {
	fields := []zap.Field{zap.String("name", cfg.Name)}
	for k, v := range cfg.Labels {
		fields = append(fields, zap.String(k, v))
	}

	proc := &process{
		outputLog:     logger.NewCompositeLogger(),
		outputBacklog: newRingBuffer(backlogBytes),

		supervisorID: supervisorID,
		config:       cfg,
		log:          zap.L().With(fields...),
		state:        Stopped,
		retryTimes:   new(int32),
	}
	proc.addToCron()
	return proc
}

func (p *process) info() *ProcessInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &ProcessInfo{
		Name:        p.config.Name,
		Description: p.description(),
		Start:       p.startTime,
		Stop:        p.getStopTime(),
		State:       p.state,
		ExitStatus:  p.exitStatus(),
		Logfile:     p.config.LogFile,
		Pid:         p.pid(),
	}
}

// matchLabels sees if a Process's label-set matches some search set.
func (p *process) matchLabels(labels map[string]string) bool {
	if len(labels) == 0 {
		return true
	}

	cfg := p.config
	if len(cfg.Labels) == 0 {
		return false
	}
	for k, v := range labels {
		tv, ok := cfg.Labels[k]
		if !ok || tv != v {
			return false
		}
	}
	return true
}

var cronParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func (p *process) updateConfig(config *config.Program) {
	// TODO it looks like this won't restart the process if attributes change
	// like the command to run, its environment, etc.
	p.cfgMu.Lock()
	p.config = config
	p.cfgMu.Unlock()
}

// add this process to crontab
func (p *process) addToCron() {
	cfg := p.config
	if cfg.Cron == "" {
		return
	}
	schedule, err := cronParser.Parse(cfg.Cron)
	if err != nil {
		p.log.Error("Invalid cron entry", zap.String("cron", cfg.Cron), zap.Error(err))
		return
	}

	p.log.Info("Scheduling process with cron", zap.String("cron", cfg.Cron))
	id := scheduler.Schedule(schedule, cron.FuncJob(func() {
		p.log.Debug("Running scheduled process")
		if !p.isRunning() {
			p.start(false)
		}
	}))

	p.cronID = id
}

func (p *process) removeFromCron() {
	if p.config != nil {
		s := p.config.Cron
		if len(s) == 0 {
			return
		}
	}

	p.log.Info("Removing process from cron schedule")
	scheduler.Remove(p.cronID)
	p.cronID = 0
}

// start start the process
// Args:
//  wait - true, wait the program started or failed
func (p *process) start(wait bool) {
	p.log.Info("Starting process")
	p.mu.Lock()
	if p.inStart {
		p.log.Info("Process already starting")
		p.mu.Unlock()
		return
	}

	p.inStart = true
	p.stopByUser = false
	p.mu.Unlock()

	signalWaiter := func() {}
	waitCh := make(chan struct{})
	if wait {
		var once sync.Once
		signalWaiter = func() {
			once.Do(func() {
				close(waitCh)
			})
		}
	} else {
		close(waitCh)
	}

	go func() {
		defer func() {
			p.mu.Lock()
			p.inStart = false
			p.mu.Unlock()
			signalWaiter()
		}()

		for {
			p.run(signalWaiter)

			// avoid print too many logs if fail to start program too quickly
			if time.Now().Unix()-p.startTime.Unix() < 2 {
				time.Sleep(5 * time.Second)
			}

			if p.stopByUser {
				p.log.Info("Stopped by user, don't start it again")
				return
			}
			if !p.isAutoRestart() {
				p.log.Info("Auto restart disabled; won't restart")
				return
			}
		}
	}()

	<-waitCh
}

// destroy stops the process and removes it from cron
func (p *process) destroy(wait bool) {
	p.removeFromCron()
	p.stop(wait)
	p.outputLog.Close()
}

// description get the process status description
func (p *process) description() string {
	if p.state == Running {
		seconds := int(time.Now().Sub(p.startTime).Seconds())
		minutes := seconds / 60
		hours := minutes / 60
		days := hours / 24
		if days > 0 {
			return fmt.Sprintf("pid %d, uptime %d days, %d:%02d:%02d", p.cmd.Process.Pid, days, hours%24, minutes%60, seconds%60)
		}
		return fmt.Sprintf("pid %d, uptime %d:%02d:%02d", p.cmd.Process.Pid, hours%24, minutes%60, seconds%60)
	} else if p.state != Stopped {
		return p.stopTime.String()
	}
	return ""
}

// GetExitStatus get the exit status of the process if the program exit
func (p *process) exitStatus() int {
	if p.state == Exited || p.state == Backoff {
		if p.cmd.ProcessState == nil {
			return 0
		}
		status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
		if ok {
			return status.ExitStatus()
		}
	}
	return 0
}

// pid returns the pid of running process or 0 it is not in running status
func (p *process) pid() int {
	switch p.state {
	case Starting, Running, Stopping:
		return p.cmd.Process.Pid
	}
	return 0
}

// stopTime returns the time the process stopped.
func (p *process) getStopTime() time.Time {
	switch p.state {
	case Starting, Running, Stopping:
		return time.Unix(0, 0)
	default:
		return p.stopTime
	}
}

// sendProcessStdin send data to process stdin
func (p *process) sendProcessStdin(chars string) error {
	if p.stdin != nil {
		_, err := p.stdin.Write([]byte(chars))
		return err
	}
	return fmt.Errorf("NO_FILE")
}

// check if the process should be
func (p *process) isAutoRestart() bool {
	restart := p.config.AutoRestart
	if restart == nil {
		if p.cmd != nil && p.cmd.ProcessState != nil {
			exitCode, err := p.getExitCode()
			// If unexpected, the process will be restarted when the program exits
			// with an exit code that is not one of the exit codes associated with
			// this process’ configuration (see exitcodes).
			return err == nil && !p.inExitCodes(exitCode)
		}
		return false
	}
	return *restart
}

func (p *process) inExitCodes(exitCode int) bool {
	for _, code := range p.config.ExitCodes {
		if code == exitCode {
			return true
		}
	}
	return false
}

func (p *process) getExitCode() (int, error) {
	if p.cmd.ProcessState == nil {
		return -1, fmt.Errorf("no exit code")
	}
	return p.cmd.ProcessState.ExitCode(), nil
}

// check if the process is running or not
//
func (p *process) isRunning() bool {
	if p.cmd != nil && p.cmd.Process != nil {
		if runtime.GOOS == "windows" {
			proc, err := os.FindProcess(p.cmd.Process.Pid)
			return proc != nil && err == nil
		}
		return p.cmd.Process.Signal(syscall.Signal(0)) == nil
	}
	return false
}

// create Command object for the program
func (p *process) createProgramCommand() error {
	cfg := p.config

	p.cmd = exec.Command(cfg.Shell, "-c", cfg.Command)

	zap.L().Info(fmt.Sprintf("creating command: %q %q %q", p.config.Shell, "-c", cfg.Command))
	p.cmd.SysProcAttr = &syscall.SysProcAttr{}
	if p.setUser() != nil {
		p.log.Error("Failed to run as user", zap.String("user", cfg.User))
		return fmt.Errorf("failed to set user")
	}
	setDeathsig(p.cmd.SysProcAttr)
	p.setEnv()
	p.setDir()
	p.setLog()

	p.log.Info("created program command")

	p.stdin, _ = p.cmd.StdinPipe()
	return nil
}

// wait for the started program exit
func (p *process) waitForExit() {
	p.cmd.Wait()
	if p.cmd.ProcessState != nil {
		p.log.Info("Process stopped", zap.Stringer("status", p.cmd.ProcessState))
	} else {
		p.log.Info("Process stopped")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopTime = time.Now().Round(time.Millisecond)
	p.currentOutputLog.Close()
}

// fail to start the program
func (p *process) failToStartProgram(reason string, finishedFn func()) {
	p.log.Error(reason)
	p.changeStateTo(Fatal)
	finishedFn()
}

// monitor if the program is in running before endTime
//
func (p *process) monitorProgramIsRunning(endTime time.Time, monitorExited, programExited *int32) {
	// if time is not expired
	for time.Now().Before(endTime) && atomic.LoadInt32(programExited) == 0 {
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
	atomic.StoreInt32(monitorExited, 1)

	p.mu.Lock()
	defer p.mu.Unlock()
	// if the program does not exit
	if atomic.LoadInt32(programExited) == 0 && p.state == Starting {
		p.log.Info("Successfully started process")
		p.changeStateTo(Running)
	}
}

func (p *process) run(finishedFn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg := p.config

	// check if the program is in running state
	if p.isRunning() {
		p.log.Info("Process already running")
		finishedFn()
		return

	}
	p.startTime = time.Now().Round(time.Millisecond)
	atomic.StoreInt32(p.retryTimes, 0)
	startSecs := cfg.StartSeconds.D
	restartPause := cfg.RestartPause.D

	var once sync.Once
	finishedOnceFn := func() {
		once.Do(finishedFn)
	}

	// process is not expired and not stopped by user
	for !p.stopByUser {
		if restartPause > 0 && atomic.LoadInt32(p.retryTimes) != 0 {
			// pause
			p.mu.Unlock()
			p.log.Info("Delay process restart", zap.Duration("restart_pause_seconds", time.Duration(restartPause)))
			time.Sleep(time.Duration(restartPause))
			p.mu.Lock()
		}
		endTime := time.Now().Add(time.Duration(startSecs))
		p.changeStateTo(Starting)
		atomic.AddInt32(p.retryTimes, 1)

		err := p.createProgramCommand()
		if err != nil {
			p.failToStartProgram("Failed to create process", finishedOnceFn)
			break
		}

		err = p.cmd.Start()

		if err != nil {
			if atomic.LoadInt32(p.retryTimes) >= int32(cfg.StartRetries) {
				p.failToStartProgram(fmt.Sprintf("fail to start process with error:%v", err), finishedOnceFn)
				break
			} else {
				p.log.Error("Failed to start process", zap.Error(err))
				p.changeStateTo(Backoff)
				continue
			}
		}

		monitorExited := int32(0)
		programExited := int32(0)
		// Set startsec to 0 to indicate that the program needn't stay
		// running for any particular amount of time.
		if startSecs <= 0 {
			p.log.Info("Process started")
			p.changeStateTo(Running)
			go finishedOnceFn()
		} else {
			go func() {
				p.monitorProgramIsRunning(endTime, &monitorExited, &programExited)
				finishedOnceFn()
			}()
		}
		p.log.Debug("Waiting for process to exit")
		p.mu.Unlock()
		p.waitForExit()

		atomic.StoreInt32(&programExited, 1)
		// wait for monitor thread exit
		for atomic.LoadInt32(&monitorExited) == 0 {
			time.Sleep(time.Duration(10) * time.Millisecond)
		}

		p.mu.Lock()

		// if the program still in running after startSecs
		if p.state == Running {
			p.changeStateTo(Exited)
			p.log.Info("Process exited")
			break
		} else {
			p.changeStateTo(Backoff)
		}

		// The number of serial failure attempts that gopm will allow when attempting to
		// start the program before giving up and putting the process into an Fatal state
		// first start time is not the retry time
		if atomic.LoadInt32(p.retryTimes) >= int32(cfg.StartRetries) {
			p.failToStartProgram(fmt.Sprintf("Unable to run process; exceeded retry count: %d", cfg.StartRetries), finishedOnceFn)
			break
		}
	}
}

func (p *process) changeStateTo(procState State) {
	p.state = procState
}

// signal sends the given signal to the process and its children.
func (p *process) signal(sig os.Signal) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.sendSignal(sig)
}

// send signal to the process
//
// Args:
//    sig - the signal to be sent
//    sigChildren - true if the signal also need to be sent to children process
//
func (p *process) sendSignal(sig os.Signal) error {
	if p.cmd != nil && p.cmd.Process != nil {
		err := signals.Kill(p.cmd.Process, sig, true)
		return err
	}
	return fmt.Errorf("process not started")
}

func (p *process) setEnv() {
	var env []string
	for k, v := range p.config.Environment {
		env = append(env, k+"="+v)
	}

	if len(env) != 0 {
		p.cmd.Env = append(env, os.Environ()...)
	} else {
		p.cmd.Env = os.Environ()
	}
}

func (p *process) setDir() {
	dir := p.config.Directory

	_, err := os.Stat(dir)
	if err != nil {
		p.log.Error("Directory does not exist.", zap.String("directory", dir))
		return
	}

	if dir != "" {
		p.cmd.Dir = dir
	}
}

func (p *process) setLog() {
	cfg := p.config

	// Remove the current loggers.
	p.outputLog.RemoveLogger(p.currentOutputLog)

	// Create a new stdout and stderr loggers using the most up-to-date
	// configuration and attach them to the composite logger.
	p.currentOutputLog = p.createLogger(cfg.LogFile, int64(cfg.LogFileMaxBytes), cfg.LogfileBackups)
	p.outputLog.AddLogger(p.currentOutputLog)

	// Attach the loggers and the backlogs to the command.
	p.cmd.Stdout = io.MultiWriter(p.outputLog, p.outputBacklog)
	p.cmd.Stderr = p.cmd.Stdout
}

func (p *process) createLogger(logFile string, maxBytes int64, backups int) logger.Logger {
	return logger.NewLogger(p.config.Name, logFile, maxBytes, backups)
}

func (p *process) setUser() error {
	userName := p.config.User
	if len(userName) == 0 {
		return nil
	}

	// check if group is provided
	pos := strings.Index(userName, ":")
	groupName := ""
	if pos != -1 {
		groupName = userName[pos+1:]
		userName = userName[0:pos]
	}
	u, err := user.Lookup(userName)
	if err != nil {
		return err
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return err
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil && groupName == "" {
		return err
	}
	if groupName != "" {
		g, err := user.LookupGroup(groupName)
		if err != nil {
			return err
		}
		gid, err = strconv.ParseUint(g.Gid, 10, 32)
		if err != nil {
			return err
		}
	}
	setUserID(p.cmd.SysProcAttr, uint32(uid), uint32(gid))
	return nil
}

// stop send signal to process to stop it
func (p *process) stop(wait bool) {
	p.mu.Lock()
	p.stopByUser = true
	isRunning := p.isRunning()
	p.mu.Unlock()
	if !isRunning {
		return
	}
	p.log.Info("Stopping process")

	cfg := p.config

	sigs := cfg.StopSignals
	if len(sigs) == 0 {
		p.log.Error("Missing signals; defaulting to KILL")
		sigs = []string{"KILL"}
	}

	waitDur := cfg.StopWaitSeconds.D

	ch := make(chan struct{})
	go func() {
		defer close(ch)

		for i := 0; i < len(sigs); i++ {
			// send signal to process
			sig, err := signals.ToSignal(sigs[i])
			if err != nil {
				continue
			}

			p.log.Info("Send stop signal to process", zap.String("signal", sigs[i]))
			_ = p.signal(sig)

			endTime := time.Now().Add(waitDur)
			for endTime.After(time.Now()) {
				// if it already exits
				if p.state != Starting && p.state != Running && p.state != Stopping {
					p.log.Info("Process exited")
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}

		p.log.Info("Process did not stop in time, sending KILL")
		p.signal(syscall.SIGKILL)
	}()

	if wait {
		<-ch
	}
}

// getStatus get the status of program in string
func (p *process) getStatus() string {
	if p.cmd.ProcessState.Exited() {
		return p.cmd.ProcessState.String()
	}
	return "running"
}
