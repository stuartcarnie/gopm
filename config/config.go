package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"github.com/robfig/cron/v3"
)

var (
	pathWithDefaults = cue.MakePath(cue.Def("#WithDefaults"))
	pathRuntime      = cue.MakePath(cue.Str("runtime"))
	pathConfig       = cue.MakePath(cue.Str("config"))
)

type options struct {
	tags []string
}

type optionFunc func(o *options)

// WithTags returns an option function to provide a list of tag values when loading the configuration.
func WithTags(tags []string) optionFunc {
	return func(o *options) {
		o.tags = tags
	}
}

// Load loads the configuration at the given directory, using the specified options, and returns it.
// On error, the error value may contain an Error value
// containing multiple errors.
func Load(configDir string, opts ...optionFunc) (*Config, error) {
	cfg, err := load0(configDir, opts...)
	if err == nil {
		return cfg, nil
	}
	if errors.As(err, new(errors.Error)) {
		err = &ConfigError{
			err: err,
		}
	}
	return nil, err
}

func load0(configDir string, opts ...optionFunc) (*Config, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

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
		Dir:  configDir,
		Tags: o.tags,
	})
	for _, inst := range insts {
		if err := inst.Err; err != nil {
			return nil, fmt.Errorf("cannot load CUE instances in %q: %w", configDir, err)
		}
	}
	vals, err := ctx.BuildInstances(insts)
	if err != nil {
		return nil, fmt.Errorf("cannot build instances: %w", err)
	}
	if len(vals) != 1 {
		return nil, fmt.Errorf("wrong value count")
	}
	val := vals[0]
	if err := val.Err(); err != nil {
		return nil, fmt.Errorf("cannot build configuration: %w", err)
	}
	// Make sure the config value is there before we fill it in with
	// our own schema.
	if val := val.LookupPath(pathConfig); val.Err() != nil {
		return nil, fmt.Errorf("cannot get \"gopm\" value containing configuration: %w", val.Err())
	}

	// Load the schema and defaults from our embedded CUE file (see schema.cue).
	lschema, err := getLocalSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get local schema: %w", err)
	}
	// Unify the user's configuration with the schema.
	val = val.Unify(lschema)

	// Fill in the runtime config, which should complete everything.
	val = val.FillPath(pathRuntime, runtime)

	// Check that it's all OK.
	if err := val.Validate(
		cue.Attributes(true),
		cue.Definitions(true),
		cue.Hidden(true),
		cue.Concrete(true),
	); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Get completed configuration.
	// Note: do this after validation because if we do it before,
	// we can hide useful error messages.
	val = val.LookupPath(pathConfig)

	// Export to JSON, then reimport, so we can apply our own defaults without
	// conflicting with any defaults applied by the user's configuration.
	data, err := val.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("cannot marshal: %w", err)
	}
	val = ctx.Encode(json.RawMessage(data))
	if err := val.Err(); err != nil {
		return nil, fmt.Errorf("cannot reencode configuration as CUE: %w", err)
	}

	// Get the local defaults and apply the completed config to them.
	val = lschema.LookupPath(pathWithDefaults).FillPath(pathConfig, val)
	if err := val.Err(); err != nil {
		return nil, fmt.Errorf("cannot fill out defaults: %w", err)
	}
	val = val.FillPath(pathRuntime, runtime)
	if err := val.Err(); err != nil {
		return nil, fmt.Errorf("cannot fill runtime: %w", err)
	}

	// Decode into the final Config value.
	var cfg Config
	if err := val.LookupPath(pathConfig).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("cannot decode into config: %w", err)
	}

	// TODO check for path conflicts in files (or we could do that in the schema, I guess,
	// although the error report would be more opaque).

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

func dumpCUE(what string, val cue.Value) {
	node := val.Syntax(cue.Hidden(true), cue.Docs(true), cue.Optional(true), cue.Definitions(true), cue.Attributes(true))
	data, err := format.Node(node, format.TabIndent(true))
	if err != nil {
		log.Printf("%s: cannot dump: %v", what, node)
		return
	}
	log.Printf("%s:\n%s\n", what, data)
}

//go:embed schema.cue
var schema []byte

// getLocalSchema returns the schema info from schema.cue.
func getLocalSchema(ctx *cue.Context) (cue.Value, error) {
	insts := load.Instances([]string{"/schema.cue"}, &load.Config{
		Overlay: map[string]load.Source{
			"/schema.cue": load.FromBytes(schema),
		},
	})
	vals, err := ctx.BuildInstances(insts)
	if err != nil {
		return cue.Value{}, fmt.Errorf("cannot build instances for local schema: %w", err)
	}
	if len(vals) != 1 {
		return cue.Value{}, fmt.Errorf("expected 1 value, got %d", len(insts))
	}
	return vals[0], nil
}

// Config defines the gopm configuration data.
type Config struct {
	// Root holds the root of the filesystem.
	Root string `json:"root"`

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
	Cron                    *CronSchedule     `json:"cron,omitempty"`
	AutoStart               bool              `json:"auto_start"`
	AutoRestart             *bool             `json:"auto_restart,omitempty"`
	RestartDirectoryMonitor string            `json:"restart_directory_monitor"`
	RestartFilePattern      string            `json:"restart_file_pattern"`
	StopSignals             []Signal          `json:"stop_signals,omitempty"`
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

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.D.String())
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
	// Note: the below options are equivalent to the cron.WithSeconds
	// scheduler option.
	parser := cron.NewParser(
		cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)
	schedule, err := parser.Parse(s)
	if err != nil {
		return fmt.Errorf("cannot parse cron entry: %v", err)
	}
	sched.Schedule = schedule
	sched.String = s
	return nil
}

func (sched *CronSchedule) MarshalJSON() ([]byte, error) {
	return json.Marshal(sched.String)
}

type ConfigError struct {
	err error
}

func (err *ConfigError) Error() string {
	return err.err.Error()
}

// AllErrors returns information on all the errors encountered
// when parsing the configuration. It returns the empty string
// if the error wasn't because of parsing the config.
func (err *ConfigError) AllErrors() string {
	var buf strings.Builder
	var cueErr errors.Error
	if !errors.As(err.err, &cueErr) {
		return ""
	}
	errors.Print(&buf, cueErr, nil)
	return buf.String()
}

func (err *ConfigError) Unwrap() error {
	return err.err
}
