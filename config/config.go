package config

import (
	"encoding/json"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func Load(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Runtime.Root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		cfg.Runtime.Root = cwd
	}
	// TODO check for path conflicts in files.
	return &cfg, nil
}

type Config struct {
	Environment map[string]string   `yaml:"environment"`
	HTTPServer  *Server             `yaml:"http_server"`
	GRPCServer  *Server             `yaml:"grpc_server"`
	Programs    map[string]*Program `yaml:"programs"`
	FileSystem  map[string]*File    `yaml:"filesystem"`
	Runtime     RuntimeConfig       `yaml:"runtime"`
}

type File struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path"`
	Content string `yaml:"content"`
	// FullPath is filled in later
}

type RuntimeConfig struct {
	Environment map[string]string `json:"environment"`
	Root        string            `json:"root"`
	CWD         string            `json:"cwd"`
}

type Program struct {
	Name                     string            `yaml:"name"`
	Directory                string            `yaml:"directory"`
	Command                  string            `yaml:"command"`
	Environment              map[string]string `yaml:"environment"`
	User                     string            `yaml:"user"`
	ExitCodes                []int             `yaml:"exit_codes" default:"[0,2]"`
	Priority                 int               `yaml:"priority" default:"999"`
	RestartPause             Duration          `yaml:"restart_pause"`
	StartRetries             int               `yaml:"start_retries" default:"3"`
	StartSeconds             Duration          `yaml:"start_seconds" default:"\"1s\""`
	Cron                     string            `yaml:"cron"`
	AutoStart                bool              `yaml:"auto_start" default:"true"`
	AutoRestart              *bool             `yaml:"auto_restart"`
	RestartDirectoryMonitor  string            `yaml:"restart_directory_monitor"`
	RestartFilePattern       string            `yaml:"restart_file_pattern" default:"*"`
	RestartWhenBinaryChanged bool              `yaml:"restart_when_binary_changed"`
	StopSignals              []string          `yaml:"stop_signals"`
	StopWaitSeconds          Duration          `yaml:"stop_wait_seconds" default:"\"10s\""`
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

type Server struct {
	Address  string `yaml:"address"`
	Network  string `yaml:"network"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Duration struct {
	D time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.D = v
	return nil
}

func (d *Duration) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.D = v
	return nil
}
