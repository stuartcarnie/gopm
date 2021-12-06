package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ochinchina/filechangemonitor"
	"github.com/robfig/cron/v3"
	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/logger"
	"github.com/stuartcarnie/gopm/signals"
	"go.uber.org/zap"
)

var gShellArgs []string

func SetShellArgs(s []string) {
	gShellArgs = s
}

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

// Process the program process management data
type Process struct {
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
	OutputLog        *logger.CompositeLogger
	currentOutputLog logger.Logger
	OutputBacklog    *RingBuffer
	cfgMu            sync.RWMutex // protects config access
	config           *config.Program
}

// NewProcess create a new Process
func NewProcess(supervisorID string, cfg *config.Program) *Process {
	fields := []zap.Field{zap.String("name", cfg.Name)}
	for k, v := range cfg.Labels {
		fields = append(fields, zap.String(k, v))
	}

	proc := &Process{
		OutputLog:     logger.NewCompositeLogger(),
		OutputBacklog: NewRingBuffer(backlogBytes),

		supervisorID: supervisorID,
		config:       cfg,
		log:          zap.L().With(fields...),
		state:        Stopped,
		retryTimes:   new(int32),
	}
	proc.addToCron()
	return proc
}

// MatchLabels sees if a Process's label-set matches some search set.
func (p *Process) MatchLabels(labels map[string]string) bool {
	if len(labels) == 0 {
		return true
	}

	cfg := p.Config()
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

func (p *Process) UpdateConfig(config *config.Program) {
	// TODO it looks like this won't restart the process if attributes change
	// like the command to run, its environment, etc.
	p.cfgMu.Lock()
	p.config = config
	p.cfgMu.Unlock()
}

func (p *Process) Config() *config.Program {
	p.cfgMu.RLock()
	defer p.cfgMu.RUnlock()
	return p.config
}

// add this process to crontab
func (p *Process) addToCron() {
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
			p.Start(false)
		}
	}))

	p.cronID = id
}

func (p *Process) removeFromCron() {
	if p.Config() != nil {
		s := p.Config().Cron
		if len(s) == 0 {
			return
		}
	}

	p.log.Info("Removing process from cron schedule")
	scheduler.Remove(p.cronID)
	p.cronID = 0
}

// Start start the process
// Args:
//  wait - true, wait the program started or failed
func (p *Process) Start(wait bool) {
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

// Destroy stops the process and removes it from cron
func (p *Process) Destroy(wait bool) {
	p.removeFromCron()
	p.Stop(wait)
	p.OutputLog.Close()
}

// Name returns the name of program
func (p *Process) Name() string {
	return p.Config().Name
}

// Description get the process status description
func (p *Process) Description() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
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
func (p *Process) ExitStatus() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

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

// GetPid get the pid of running process or 0 it is not in running status
func (p *Process) Pid() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	switch p.state {
	case Starting, Running, Stopping:
		return p.cmd.Process.Pid
	}
	return 0
}

// GetState Get the process state
func (p *Process) State() State {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.state
}

// GetStartTime get the process start time
func (p *Process) StartTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.startTime
}

// GetStopTime get the process stop time
func (p *Process) StopTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()

	switch p.state {
	case Starting:
		fallthrough
	case Running:
		fallthrough
	case Stopping:
		return time.Unix(0, 0)
	default:
		return p.stopTime
	}
}

// Logfile returns the program's output log file
func (p *Process) Logfile() string {
	return p.Config().LogFile
}

// SendProcessStdin send data to process stdin
func (p *Process) SendProcessStdin(chars string) error {
	if p.stdin != nil {
		_, err := p.stdin.Write([]byte(chars))
		return err
	}
	return fmt.Errorf("NO_FILE")
}

// check if the process should be
func (p *Process) isAutoRestart() bool {
	restart := p.Config().AutoRestart
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

func (p *Process) inExitCodes(exitCode int) bool {
	for _, code := range p.Config().ExitCodes {
		if code == exitCode {
			return true
		}
	}
	return false
}

func (p *Process) getExitCode() (int, error) {
	if p.cmd.ProcessState == nil {
		return -1, fmt.Errorf("no exit code")
	}
	return p.cmd.ProcessState.ExitCode(), nil
}

// check if the process is running or not
//
func (p *Process) isRunning() bool {
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
func (p *Process) createProgramCommand() error {
	cfg := p.Config()

	args := strings.SplitN(cfg.Command, " ", 2)

	p.cmd = exec.Command(gShellArgs[0], append(gShellArgs[1:], cfg.Command)...)
	zap.L().Info(fmt.Sprintf("creating command: %q %q", gShellArgs[0], append(gShellArgs[1:], cfg.Command)))
	p.cmd.SysProcAttr = &syscall.SysProcAttr{}
	if p.setUser() != nil {
		p.log.Error("Failed to run as user", zap.String("user", cfg.User))
		return fmt.Errorf("failed to set user")
	}
	setDeathsig(p.cmd.SysProcAttr)
	p.setEnv()
	p.setDir()
	p.setLog()
	// TODO(BUG) This assumes that the binary is directly inside the command's directory,
	// but it may be anywhere else in $PATH instead.
	bin := filepath.Join(p.cmd.Dir, args[0])
	p.setProgramRestartChangeMonitor(bin)

	p.log.Info("created program command")

	p.stdin, _ = p.cmd.StdinPipe()
	return nil
}

func (p *Process) setProgramRestartChangeMonitor(programPath string) {
	cfg := p.Config()

	if cfg.RestartWhenBinaryChanged {
		absPath, err := filepath.Abs(programPath)
		if err != nil {
			absPath = programPath
		}
		AddProgramChangeMonitor(absPath, func(path string, mode filechangemonitor.FileChangeMode) {
			p.log.Info("Process binary changed")
			p.Stop(true)
			p.Start(true)
		})
	}
	dirMonitor := cfg.RestartDirectoryMonitor
	filePattern := cfg.RestartFilePattern
	if dirMonitor != "" {
		absDir, err := filepath.Abs(dirMonitor)
		if err != nil {
			absDir = dirMonitor
		}
		AddConfigChangeMonitor(absDir, filePattern, func(path string, mode filechangemonitor.FileChangeMode) {
			// fmt.Printf( "filePattern=%s, base=%s\n", filePattern, filepath.Base( path ) )
			// if matched, err := filepath.Match( filePattern, filepath.Base( path ) ); matched && err == nil {
			p.log.Info("Watched file for process has changed")
			p.Stop(true)
			p.Start(true)
			//}
		})
	}
}

// wait for the started program exit
func (p *Process) waitForExit() {
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
func (p *Process) failToStartProgram(reason string, finishedFn func()) {
	p.log.Error(reason)
	p.changeStateTo(Fatal)
	finishedFn()
}

// monitor if the program is in running before endTime
//
func (p *Process) monitorProgramIsRunning(endTime time.Time, monitorExited, programExited *int32) {
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

func (p *Process) run(finishedFn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg := p.Config()

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

func (p *Process) changeStateTo(procState State) {
	p.state = procState
}

// Signal sends the given signal to the process and its children.
func (p *Process) Signal(sig os.Signal) error {
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
func (p *Process) sendSignal(sig os.Signal) error {
	if p.cmd != nil && p.cmd.Process != nil {
		err := signals.Kill(p.cmd.Process, sig, true)
		return err
	}
	return fmt.Errorf("process not started")
}

func (p *Process) setEnv() {
	var env []string
	for k, v := range p.Config().Environment {
		env = append(env, k+"="+v)
	}

	if len(env) != 0 {
		p.cmd.Env = append(env, os.Environ()...)
	} else {
		p.cmd.Env = os.Environ()
	}
}

func (p *Process) setDir() {
	dir := p.Config().Directory

	_, err := os.Stat(dir)
	if err != nil {
		p.log.Error("Directory does not exist.", zap.String("directory", dir))
		return
	}

	if dir != "" {
		p.cmd.Dir = dir
	}
}

func (p *Process) setLog() {
	cfg := p.Config()

	// Remove the current loggers.
	p.OutputLog.RemoveLogger(p.currentOutputLog)

	// Create a new stdout and stderr loggers using the most up-to-date
	// configuration and attach them to the composite logger.
	p.currentOutputLog = p.createLogger(cfg.LogFile, int64(cfg.LogFileMaxBytes), cfg.LogfileBackups)
	p.OutputLog.AddLogger(p.currentOutputLog)

	// Attach the loggers and the backlogs to the command.
	p.cmd.Stdout = io.MultiWriter(p.OutputLog, p.OutputBacklog)
	p.cmd.Stderr = p.cmd.Stdout
}

func (p *Process) createLogger(logFile string, maxBytes int64, backups int) logger.Logger {
	return logger.NewLogger(p.Name(), logFile, maxBytes, backups)
}

func (p *Process) setUser() error {
	userName := p.Config().User
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

// Stop send signal to process to stop it
func (p *Process) Stop(wait bool) {
	p.mu.Lock()
	p.stopByUser = true
	isRunning := p.isRunning()
	p.mu.Unlock()
	if !isRunning {
		return
	}
	p.log.Info("Stopping process")

	cfg := p.Config()

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
			_ = p.Signal(sig)

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
		p.Signal(syscall.SIGKILL)
	}()

	if wait {
		<-ch
	}
}

// GetStatus get the status of program in string
func (p *Process) GetStatus() string {
	if p.cmd.ProcessState.Exited() {
		return p.cmd.ProcessState.String()
	}
	return "running"
}
