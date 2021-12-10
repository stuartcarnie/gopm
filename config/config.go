package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/robfig/cron/v3"

	"github.com/stuartcarnie/gopm/signals"
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
	if !filepath.IsAbs(configDir) {
		configDir = filepath.Join(wd, configDir)
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
		Dir: configDir,
	})
	for _, inst := range insts {
		if err := inst.Err; err != nil {
			// TODO print to log file instead of directly to stderr.
			errors.Print(os.Stderr, err, nil)
			return nil, fmt.Errorf("cannot load CUE instances in %q: %v", configDir, err)
		}
	}
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

	if err := cfg.verifyDependencies(); err != nil {
		return nil, err
	}
	if cfg.GRPCServer == nil && cfg.HTTPServer == nil {
		return nil, fmt.Errorf("configuration in %q must specify at least one of grpc_server or http_server", configDir)
	}
	return &cfg, nil
}

// verifiyDependencies verifies the program depends_on fields
// and also populates the TopoSortedPrograms field.
func (cfg *Config) verifyDependencies() error {
	if err := cfg.checkDepsExist(); err != nil {
		return err
	}
	sorted, cycles := topoSort(cfg.Programs)
	if len(cycles) > 0 {
		return fmt.Errorf("cycles detected in program dependencies: %v", dumpCycles(cycles))
	}
	// This is a horrible O(>n^2) algorithm but n is gonna be very small.
	// What's the name for what this is doing anyway?
	var sortedProgs [][]*Program
	for i := 0; i < len(sorted); i++ {
		if sorted[i] == "" {
			// It's already been taken.
			continue
		}
		cohort := make([]*Program, 0, 1)
		cohortSet := make(map[*Program]bool)
		// Take all the programs that don't have dependencies on
		// the current cohort.
		for j := i; j < len(sorted); j++ {
			if sorted[j] == "" {
				continue
			}
			p := cfg.Programs[sorted[j]]
			if !cfg.dependsOn(p, cohortSet) {
				cohort = append(cohort, p)
				cohortSet[p] = true
				sorted[j] = ""
			}
		}
		// Sort for consistency.
		sort.Slice(cohort, func(i, j int) bool {
			return cohort[i].Name < cohort[j].Name
		})
		sortedProgs = append(sortedProgs, cohort)
	}
	cfg.TopoSortedPrograms = sortedProgs
	return nil
}

func (cfg *Config) checkDepsExist() error {
	for _, p := range cfg.Programs {
		for _, dep := range p.DependsOn {
			if cfg.Programs[dep] == nil {
				return fmt.Errorf("program %q has dependency on non-existent program %q", p.Name, dep)
			}
		}
	}
	return nil
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

	// TopoSortedPrograms holds all the programs in topologically
	// sorted order (children before parents). It's populated
	// after reading the configuration.
	// Each element in the slice a slice of independent programs:
	// that is, all the programs in TopoSortedPrograms[i] must
	// started before all the programs in TopoSortedPrograms[i+1]
	// but there's no relationship between them.
	TopoSortedPrograms [][]*Program `json:"-"`
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
	Name                    string            `json:"name"`
	Directory               string            `json:"directory"`
	Command                 string            `json:"command"`
	Description             string            `json:"description,omitempty"`
	Shell                   string            `json:"shell"`
	Environment             map[string]string `json:"environment"`
	User                    string            `json:"user"`
	ExitCodes               []int             `json:"exit_codes"`
	RestartPause            Duration          `json:"restart_pause"`
	StartRetries            int               `json:"start_retries"`
	StartSeconds            Duration          `json:"start_seconds"`
	Cron                    CronSchedule      `json:"cron,omitempty"`
	AutoStart               bool              `json:"auto_start"`
	AutoRestart             *bool             `json:"auto_restart,omitempty"`
	RestartDirectoryMonitor string            `json:"restart_directory_monitor"`
	RestartFilePattern      string            `json:"restart_file_pattern"`
	StopSignals             []Signal          `json:"stop_signals"`
	StopWaitSeconds         Duration          `json:"stop_wait_seconds"`
	StopAsGroup             bool              `json:"stop_as_group"`
	KillAsGroup             bool              `json:"kill_as_group"`
	LogFile                 string            `json:"logfile"`
	LogFileBackups          int               `json:"logfile_backups"`
	LogFileMaxBytes         int64             `json:"logfile_max_bytes"`
	LogFileMaxBacklogBytes  int               `json:"logfile_max_backlog_bytes"`
	DependsOn               []string          `json:"depends_on"`
	Labels                  map[string]string `json:"labels"`
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

type CronSchedule struct {
	Schedule cron.Schedule
	String   string
}

func (sched *CronSchedule) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		sched.Schedule = nil
		sched.String = ""
		return nil
	}
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(s)
	if err != nil {
		return fmt.Errorf("cannot parse cron entry: %v", err)
	}
	sched.Schedule = schedule
	sched.String = s
	return nil
}

type Signal struct {
	S      os.Signal
	String string
}

func (s *Signal) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	sig, err := signals.ToSignal(str)
	if err != nil {
		return err
	}
	s.S = sig
	s.String = str
	return nil
}
