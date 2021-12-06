package gopm

import (
	"list"
	pathpkg "path"
	"time"
)

#Config: {
	// runtime is filled in by gopm itself and can be used to
	// fill in the rest of the configuration values.
	runtime: #RuntimeConfig

	// gprc_server specifies that a gRPC server should
	// run providing access to the running gopm programs.
	grpc_server?: #Server

	// http_server specifies that an HTTP server should
	// run providing access to the running gopm programs.
	http_server?: #Server

	// programs holds all the programs to start. Each program will only
	// be started when its dependencies have successfully started.
	programs: [string]: #Program
	programs: [name=_]: "name": name

	// filesystem holds the files to create before starting the programs.
	// It's keyed by an arbitrary key (not necessarily the file path).
	// The full filename of a given file f can be found by looking in
	// #paths[f].
	filesystem: [string]: #File
	filesystem: [name=_]: "name": name

	// #paths holds an entry for each file, making it easier for a configuration
	// to find out the full path name of any file above.
	#Paths: [string]: string
	#Paths: {
		for name, f in filesystem {
			// The string interpolation below is to work around https://github.com/cue-lang/cue/issues/1409
			"\(name)": pathpkg.Join(["\(runtime.root)", f.path], pathpkg.Unix)
		}
	}
}

#Server: {
	address: string & =~ "."
	network?: string
}

#RuntimeConfig: {
	// env holds all the environment variables when gopm is started.
	// TODO as an alternative, we could say that this should
	// be filled in with incomplete entries specifying what env vars
	// might be expected (which could also include constraints on their values)
	environment: [string]: string

	// root holds the root directory configured for gopm's filesystem.
	root: *cwd | string

	// cwd holds the current working directory.
	cwd: string
}

#Program: {
	name:       =~"^\\w+$"
	directory?: string
	command:    string & =~ "."
	environment?: {
		{[=~"^\\w+$"]: string}
	}
	user?: string
	exit_codes?: [...int & >=0 & <=255]
	priority?:      int
	restart_pause?: int
	start_retries?: int & >=0
	start_seconds?: time.Duration
	cron?:          string		// TODO validate this
	auto_start?:    bool
	auto_restart?:  bool

	// Path to a directory to monitor and automatically restart the
	// process if changes are detected
	restart_directory_monitor?: string

	// A pattern used to monitor a path and automatically restart the
	// process if changes are detected
	restart_file_pattern?: string

	// Automatically restart the process if changes to the binary are
	// detected
	restart_when_binary_changed?: bool
	stop_signals?:                list.UniqueItems() & [..."HUP" | "INT" | "QUIT" | "KILL" | "TERM" | "USR1" | "USR2"]

	// The time to wait for the process to stop before killing.
	stop_wait_seconds?:        time.Duration
	stop_as_group?:            bool
	kill_as_group?:            bool

	// A list of process names that must be started before starting
	// this process
	depends_on?: [...string]
	labels: [string]: string

	logfile?:           string
	logfile_backups?:   int
	logfile_max_bytes?: int
}

#File: {
	name:    =~"^\\w+$"
	path:    string & =~ "."
	content: string
}

#ConfigWithDefaults: #Config &  {
	...
	programs: [_]: #Program & {
		exit_codes: *[0, 2] | _
		priority: *999 | _
		start_retries: *3 | _
		start_seconds: *"1s" | _
		auto_start: *true | _
		auto_restart: *false | _
		restart_file_pattern: *"*" | _
		kill_as_group: *true | _
		stop_signals: *["INT"] | _
		stop_as_group: *true | _
		stop_wait_seconds: *"10s" | _
		logfile: *"/dev/null" | _
		logfile_backups: *10 | _
		logfile_max_bytes: *50Mi | _
		...
	}
}
