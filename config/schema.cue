package gopm

import (
	pathpkg "path"
	"regexp"
	"time"
)

// future-proofing
version: "gopm.v1"
runtime: #RuntimeConfig
config:  #Config

// #Config defines the contents of the "config" value.
#Config: {
	// root holds the root directory for the filesystem
	// files. This is also the default directory for programs
	// to run in.
	root: *runtime.cwd | string

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
	// The full filename of a given file f can be found by looking in the absPath
	// field.
	filesystem: [string]: #File
	filesystem: [name=_]: "name": name
	filesystem: [_]: {
		path:    _
		absPath: pathpkg.Join([root, path], pathpkg.Unix)
	}
}

#Server: {
	address: string & =~"."
	network: *"tcp" | string
}

#RuntimeConfig: {
	// env holds all the environment variables when gopm is started.
	// TODO as an alternative, we could say that this should
	// be filled in with incomplete entries specifying what env vars
	// might be expected (which could also include constraints on their values)
	environment: [string]: string

	// cwd holds the current working directory.
	cwd: string
}

#Program: {
	// name holds the name of the program. This is implied
	// from the name of the program entry.
	name: =~"^\\w+$"

	// description holds a human-readable description of the program.
	description?: string

	// directory holds the directory in which to run the program.
	directory?: string

	// command holds the command to run the program.
	// If it's empty, the command will always be considered
	// to succeed (the shell value is ignored in this case).
	// This can be useful for making a program that acts as
	// a pseudo-target for a bunch of other dependencies.
	command: string

	// probe specifies a way to find out when a command
	// is running. If oneshot is true, the probe is run once
	// when the command exits successfully; otherwise it's run repeatedly
	// after the command has been started until it succeeds
	// or the start_seconds timeout has been exceeded.
	probe?: {
		// command specifies a probe command to run.
		// It should return immediately and its exit status should
		// reflect whether the program command is running (0 means
		// it's running).
		//
		// The probe command is run in the same directory and with the
		// same environment as the program command.
		command: string

		// shell specifies the shell to use to run the command.
		// It defaults to the same shell as the program.
		"shell"?: string

		// retry specifies how often to run the probe
		// after starting the command.
		retry?: #ProbeRetry
	} | {
		// url specifies an HTTP URL to fetch.
		// The command is deemed to be running if a GET
		// request on the URL returns a 2xx status.
		url: string

		// retry specifies how often to run the probe
		// after starting the command.
		retry?: #ProbeRetry
	} | {
		// file specifies a file to read.
		// The program command is deemed to be running if the file exists and, if
		// pattern is provided, the file contains a match for that.
		//
		// If the path isn't absolute, it's interpreted relative to
		// the program's directory.
		file:     string
		pattern?: regexp.Valid

		// retry specifies how often to read the file after
		// starting the command.
		retry?: #ProbeRetry
	} | {
		// output specifies a pattern to match in the program's stdout
		// or stderr. The pattern cannot match multiple lines.
		output: regexp.Valid
	}

	// oneshot specifies that the command is always intended
	// to run to completion. If this is true, start_seconds is ignored and
	// the program will never enter the Running state - programs that
	// depend on it will be started only when the command exits successfully.
	oneshot?: bool

	// shell specifies the shell command to use to interpret the
	// above command. The shell is invoked as $shell -c $command.
	shell?: string

	// depends_on holds set of process names that must be started before starting
	// this process. Each key in the map must name another program.
	// Cyclic dependencies are not allowed.
	depends_on?: [string]: true

	// labels holds a set of labels to associate with the program.
	// Some gopmctl commands can be applied to all programs with
	// a matching set of labels.
	labels: [string]: string

	// environment holds the environment variables to pass to the program.
	// Note that this is empty by default.
	environment?: [string]: string

	// user holds the username to run the process as.
	user?: string

	// restart_pause holds the length of time to wait after a program
	// has exited before auto-restarting it.
	restart_pause?: time.Duration

	// start_retries holds the maximum number of times to try auto-restarting
	// a program before giving up.
	start_retries?: int & >=0

	// start_seconds is the time for which the program needs to stay running after a
	// startup to consider the start successful.
	start_seconds?: time.Duration

	// cron holds a cron schedule for running the program.
	cron?: string

	// auto_start indicates whether the program should start automatically when
	// gopm is started.
	auto_start?: bool

	// auto_restart indicates whether the program should be restarted automatically
	// after it exits. If it's not present, the program will be restarted if it exits with an
	// exit code not mentioned in exit_codes.
	auto_restart?: bool

	// exit_codes holds the set of "expected" exit codes. If the program exits
	// with one of these codes and auto_restart isn't present, it won't be restarted.
	exit_codes?: [...int & >=0 & <=255]

	// restart_directory_monitor holds a path to a directory to monitor.
	// If any changes are detected, the process will be restarted.
	restart_directory_monitor?: string

	// restart_file_pattern is a shell-style wildcard that selects files considered for restart
	// when restart_directory_monitor is set.
	restart_file_pattern?: string

	// stop_signals holds a list of signals to send in order to try to kill the running process.
	// There will be a pause of stop_wait_seconds after each attempt.
	stop_signals?: [..."HUP" | "INT" | "QUIT" | "KILL" | "TERM" | "USR1" | "USR2"]

	// stop_wait_seconds holds the time to wait for the process to stop after sending a signal.
	stop_wait_seconds?: time.Duration

	// logfile holds the file to write the output of the process to. If this is /dev/stderr
	// or /dev/stdout, the output will be written to gopm's standard error or standard
	// output respectively. If it's empty, no on-disk file will be created.
	logfile?: string

	// logfile_backups specifies how many on-disk backup files to retain.
	// beyond the most recently rolled log file. Backup files are named
	// "\(logfile).\(number)" where number starts from 1.
	logfile_backups?: int & >=0

	// When the log file gets to logfile_max_bytes in size, it's
	// moved to a backup file and a new one created.
	logfile_max_bytes?: int

	// logfile_max_backlog_bytes specifies the maximum number of
	// in-memory bytes to use for output backlog.
	logfile_max_backlog_bytes?: int
}

#ProbeRetry: {
	// delay holds the amount of time between the start of each iteration.
	// If factor is specified or max_delay is greater
	// than delay, then the maximum delay time will increase
	// exponentially (modulo jitter) as iterations continue, up to a
	// maximum of max_delay.
	delay:      time.Duration
	max_delay?: time.Duration
	factor?:    number & >=1
	regular?:   bool

	// TODO we could include max_count and max_duration
	// here to make it possible to specify that if a probe
	// hasn't succeeded after some time, it should be considered
	// to have failed (and go into Fatal state) but that isn't
	// straightforward, so only allow unlimited retries for now.
}

#File: {
	name: =~"^\\w+$"
	// path holds the path relative to the filesystem root.
	path:    string & =~"."
	content: string

	// absPath is filled in automatically - it joins
	// the filesystem root with the above path.
	absPath: string
}

#WithDefaults: {
	runtime: #RuntimeConfig

	config: #Config
	config: root: *runtime.cwd | _
	config: programs: [_]: #Program & {
		directory:            *runtime.cwd | _
		shell:                *"/bin/sh" | _
		exit_codes:           *[0, 2] | _
		start_retries:        *3 | _
		start_seconds:        *"1s" | _
		auto_start:           *true | _
		oneshot:              *false | _
		auto_restart:         *false | _
		restart_file_pattern: *"*" | _
		stop_signals:         *["INT", "KILL"] | _
		stop_wait_seconds:    *"10s" | _
		probe?:               {
			command: _
			"shell": *shell | _
			retry:   *#defaultRetry | _
			...
		} | {
			url:   _
			retry: *#defaultRetry | _
			...
		} | {
			file:  _
			retry: *#defaultRetry | _
			...
		} | {
			output: _
			...
		}
		logfile:                   *"/dev/null" | _
		logfile_backups:           *1 | _
		logfile_max_bytes:         *50Mi | _
		logfile_max_backlog_bytes: *1Mi | _
	}

	...
}

#defaultRetry: {
	delay:     "1ms"
	max_delay: "100ms"
}
