package config

import (
	"io"
	"time"

	"github.com/goccy/go-yaml"
)

func parseRoot(reader io.Reader) (*root, error) {
	dec := yaml.NewDecoder(reader)
	var v root
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

type root struct {
	Environment map[string]string `json:"environment"`
	HttpServer  *httpServer       `json:"http_server"`
	GrpcServer  *grpcServer       `json:"grpc_server"`
	Programs    []*program        `json:"programs"`
	FileSystem  map[string]*file  `json:"filesystem"`
	Runtime     runtimeConfig     `json:"runtime"`
}

type file struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type runtimeConfig struct {
	Environment map[string]string `json:"environment"`
	Root        string            `json:"root"`
	CWD         string            `json:"cwd"`
}

type program struct {
	Name                     string            `yaml:"name"`
	Directory                string            `yaml:"directory"`
	Command                  string            `yaml:"command"`
	Environment              map[string]string `yaml:"environment"`
	User                     string            `yaml:"user"`
	ExitCodes                []int             `yaml:"exit_codes" default:"[0,2]"`
	Priority                 int               `yaml:"priority" default:"999"`
	RestartPause             duration          `yaml:"restart_pause"`
	StartRetries             int               `yaml:"start_retries" default:"3"`
	StartSeconds             duration          `yaml:"start_seconds" default:"1000000000"`
	Cron                     string            `yaml:"cron"`
	AutoStart                bool              `yaml:"auto_start" default:"true"`
	AutoRestart              *bool             `yaml:"auto_restart"`
	RestartDirectoryMonitor  string            `yaml:"restart_directory_monitor"`
	RestartFilePattern       string            `yaml:"restart_file_pattern" default:"*"`
	RestartWhenBinaryChanged bool              `yaml:"restart_when_binary_changed"`
	StopSignals              []string          `yaml:"stop_signals"`
	StopWaitSeconds          duration          `yaml:"stop_wait_seconds" default:"10000000000"`
	StopAsGroup              bool              `yaml:"stop_as_group"`
	KillAsGroup              bool              `yaml:"kill_as_group"`
	StdoutLogFile            string            `yaml:"stdout_logfile" default:"/dev/null"`
	StdoutLogfileBackups     int               `yaml:"stdout_logfile_backups" default:"10"`
	StdoutLogFileMaxBytes    int               `yaml:"stdout_logfile_max_bytes" default:"52428800"`
	RedirectStderr           bool              `yaml:"redirect_stderr"`
	StderrLogFile            string            `yaml:"stderr_logfile" default:"/dev/null"`
	StderrLogfileBackups     int               `yaml:"stderr_logfile_backups" default:"10"`
	StderrLogFileMaxBytes    int               `yaml:"stderr_logfile_max_bytes" default:"52428800"`
	DependsOn                []string          `yaml:"depends_on"`
	Labels                   map[string]string `yaml:"labels"`
}

type grpcServer struct {
	Address  string `yaml:"address"`
	Network  string `yaml:"network"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type httpServer struct {
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type duration time.Duration

func (d *duration) UnmarshalYAML(bytes []byte) error {
	var s string
	if err := yaml.Unmarshal(bytes, &s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = duration(v)
	return nil
}
