package config

import (
	"time"

	"github.com/robfig/cron/v3"
)

type AutoStartMode int

const (
	AutoStartModeDefault AutoStartMode = iota
	AutoStartModeAlways
	AutoStartModeNever
)

type Process struct {
	Name                     string
	Directory                string
	Command                  string
	Environment              map[string]string
	User                     string
	ExitCodes                []int
	Priority                 int
	RestartPause             time.Duration
	StartRetries             int
	StartSeconds             time.Duration
	Cron                     string
	AutoStart                bool
	AutoRestart              AutoStartMode
	RestartDirectoryMonitor  string
	RestartFilePattern       string
	RestartWhenBinaryChanged bool
	StopSignals              []string
	StopWaitSeconds          time.Duration
	StopAsGroup              bool
	KillAsGroup              bool
	StdoutLogFile            string
	StdoutLogfileBackups     int
	StdoutLogFileMaxBytes    int
	RedirectStderr           bool
	StderrLogFile            string
	StderrLogfileBackups     int
	StderrLogFileMaxBytes    int
	DependsOn                []string
	Labels                   map[string]string
}

var cronParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func (p *Process) CronSchedule() cron.Schedule {
	if len(p.Cron) == 0 {
		return nil
	}
	s, err := cronParser.Parse(p.Cron)
	if err != nil {
		panic(err)
	}
	return s
}

type Server struct {
	Name     string
	Network  string
	Address  string
	Username string
	Password string
}
