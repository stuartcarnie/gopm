package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
)

// Load loads the configuration at the given directory and returns it.
// If root isn't empty, it configures the location of the filesystem root.
func Load(configDir string, root string) (*Config, error) {
	info, err := os.Stat(configDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		// TODO allow YAML/JSON too?
		return nil, fmt.Errorf("config %q must be a directory", configDir)
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	// runtime holds the values that are provided to the configuration
	// from which everything else derives.
	runtime := &RuntimeConfig{
		Root:        root,
		CWD:         wd,
		Environment: make(map[string]string),
	}
	for _, v := range os.Environ() {
		vv := strings.SplitN(v, "=", 2)
		runtime.Environment[vv[0]] = vv[1]
	}

	// Load the configuration, which will be incomplete initially
	// as we haven't yet unified with the runtime config.
	ctx := cuecontext.New()
	insts := load.Instances([]string{"."}, &load.Config{
		Dir: filepath.Join(wd, configDir),
	})
	vals, err := ctx.BuildInstances(insts)
	if err != nil {
		return nil, fmt.Errorf("cannot build instances: %v", err)
	}
	if len(vals) != 1 {
		return nil, fmt.Errorf("wrong value count")
	}

	// Load the schema and defaults from our embedded CUE file (see schema.cue).
	lschema, err := getLocalSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get local schema: %v", err)
	}

	// Unify the user's configuration with the schema.
	val := vals[0].Unify(lschema.config)

	// Fill in the runtime config, which should complete everything.
	val = val.FillPath(cue.ParsePath("runtime"), runtime)

	// Check that it's all OK.
	if err := val.Validate(cue.Concrete(true)); err != nil {
		// TODO print to log file instead of directly to stderr.
		errors.Print(os.Stderr, err, nil)
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	// Export to JSON, then reimport, so we can apply our own defaults without
	// conflicting with any defaults applied by the user's configuration.
	data, err := val.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("cannot marshal: %v", err)
	}

	val = ctx.Encode(json.RawMessage(data))
	if err := val.Err(); err != nil {
		return nil, fmt.Errorf("cannot reencode configuration as CUE: %v", err)
	}

	// Apply our local defaults.
	val = val.Unify(lschema.defaults)

	// TODO check for path conflicts in files (or we could do that in the schema, I guess,
	// although the error report would be more opaque).

	// Decode into the final Config value.
	var cfg Config
	if err := val.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("cannot decode into config: %v", err)
	}
	return &cfg, nil
}

type localSchema struct {
	// config holds the #Config definition
	config cue.Value
	// defaults holds the #Defaults definition.
	defaults cue.Value
}

//go:embed schema.cue
var schema []byte

// getLocalSchema returns the schema info from schema.cue.
func getLocalSchema(ctx *cue.Context) (*localSchema, error) {
	insts := load.Instances([]string{"/schema.cue"}, &load.Config{
		Overlay: map[string]load.Source{
			"/schema.cue": load.FromBytes(schema),
		},
	})
	vals, err := ctx.BuildInstances(insts)
	if err != nil {
		return nil, fmt.Errorf("cannot build instances for local schema: %v", err)
	}
	if len(vals) != 1 {
		return nil, fmt.Errorf("expected 1 value, got %d", len(insts))
	}
	val := vals[0]
	configDef := val.LookupPath(cue.MakePath(cue.Def("#Config")))
	if err := configDef.Err(); err != nil {
		return nil, fmt.Errorf("cannot get #Config: %v", err)
	}
	defaultsDef := val.LookupPath(cue.MakePath(cue.Def("#ConfigWithDefaults")))
	if err := configDef.Err(); err != nil {
		return nil, fmt.Errorf("cannot get #ConfigWithDefaults: %v", err)
	}
	return &localSchema{
		config:   configDef,
		defaults: defaultsDef,
	}, nil
}

// Config defines the gopm configuration data.
type Config struct {
	// Runtime is provided to the user in order that they
	// can fill out the rest of their configuration.
	Runtime RuntimeConfig `json:"runtime"`

	Environment map[string]string   `json:"environment"`
	HTTPServer  *Server             `json:"http_server"`
	GRPCServer  *Server             `json:"grpc_server"`
	Programs    map[string]*Program `json:"programs"`
	FileSystem  map[string]*File    `json:"filesystem"`
}

type File struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type RuntimeConfig struct {
	Environment map[string]string `json:"environment"`
	Root        string            `json:"root,omitempty"`
	CWD         string            `json:"cwd"`
}

type Program struct {
	Name                     string            `json:"name"`
	Directory                string            `json:"directory"`
	Command                  string            `json:"command"`
	Environment              map[string]string `json:"environment"`
	User                     string            `json:"user"`
	ExitCodes                []int             `json:"exit_codes"`
	Priority                 int               `json:"priority"`
	RestartPause             Duration          `json:"restart_pause"`
	StartRetries             int               `json:"start_retries"`
	StartSeconds             Duration          `json:"start_seconds"`
	Cron                     string            `json:"cron"`
	AutoStart                bool              `json:"auto_start"`
	AutoRestart              *bool             `json:"auto_restart,omitempty"`
	RestartDirectoryMonitor  string            `json:"restart_directory_monitor"`
	RestartFilePattern       string            `json:"restart_file_pattern"`
	RestartWhenBinaryChanged bool              `json:"restart_when_binary_changed"`
	StopSignals              []string          `json:"stop_signals"`
	StopWaitSeconds          Duration          `json:"stop_wait_seconds"`
	StopAsGroup              bool              `json:"stop_as_group"`
	KillAsGroup              bool              `json:"kill_as_group"`
	StdoutLogFile            string            `json:"stdout_logfile"`
	StdoutLogfileBackups     int               `json:"stdout_logfile_backups"`
	StdoutLogFileMaxBytes    int               `json:"stdout_logfile_max_bytes"`
	RedirectStderr           bool              `json:"redirect_stderr"`
	StderrLogFile            string            `json:"stderr_logfile"`
	StderrLogfileBackups     int               `json:"stderr_logfile_backups"`
	StderrLogFileMaxBytes    int               `json:"stderr_logfile_max_bytes"`
	DependsOn                []string          `json:"depends_on"`
	Labels                   map[string]string `json:"labels"`
}

type Server struct {
	Address string `json:"address"`
	Network string `json:"network"`
}

type Duration struct {
	D time.Duration
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
