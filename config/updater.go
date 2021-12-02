package config

import (
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/r3labs/diff"
	"github.com/scylladb/go-set/strset"
)

func applyUpdates(txn *memdb.Txn, m *root) error {
	var u updater
	return u.update(txn, m)
}

type updater struct{}

func (u *updater) update(txn *memdb.Txn, m *root) error {
	u.applyGroup(txn, m)
	u.applyHttpServer(txn, m)
	u.applyGrpcServer(txn, m)
	u.applyFileSystem(txn, m)
	u.applyPrograms(txn, m)
	return nil
}

func (u *updater) applyGroup(txn *memdb.Txn, m *root) {
	for _, g := range m.Groups {
		obj := &group{
			Name:     g.Name,
			Programs: g.Programs,
		}
		raw, _ := txn.First("group", "id", g.Name)
		if orig, ok := raw.(*group); ok && !diff.Changed(orig, obj) {
			continue
		}
		_ = txn.Insert("group", obj)
	}
}

func (u *updater) applyPrograms(txn *memdb.Txn, m *root) error {
	iter, err := txn.Get("process", "id")
	if err != nil {
		return err
	}

	prev := strset.New()
	for {
		raw := iter.Next()
		if orig, ok := raw.(*Process); ok {
			prev.Add(orig.Name)
			continue
		}
		break
	}

	next := strset.New()
	for _, program := range m.Programs {
		environment := map[string]string{}
		for k, v := range m.Environment {
			environment[k] = v
		}
		for k, v := range program.Environment {
			environment[k] = v
		}

		next.Add(program.Name)
		proc := &Process{
			Group:                    program.Name, // TODO(sgc): Add back groups,
			Name:                     program.Name,
			Directory:                program.Directory,
			Command:                  program.Command,
			Environment:              environment,
			User:                     program.User,
			ExitCodes:                program.ExitCodes,
			Priority:                 program.Priority,
			RestartPause:             time.Duration(program.RestartPause),
			StartRetries:             program.StartRetries,
			StartSeconds:             time.Duration(program.StartSeconds),
			Cron:                     program.Cron,
			AutoStart:                program.AutoStart,
			RestartDirectoryMonitor:  program.RestartDirectoryMonitor,
			RestartFilePattern:       program.RestartFilePattern,
			RestartWhenBinaryChanged: program.RestartWhenBinaryChanged,
			StopSignals:              program.StopSignals,
			StopWaitSeconds:          time.Duration(program.StopWaitSeconds),
			StopAsGroup:              program.StopAsGroup,
			KillAsGroup:              program.KillAsGroup,
			StdoutLogFile:            program.StdoutLogFile,
			StdoutLogfileBackups:     program.StdoutLogfileBackups,
			StdoutLogFileMaxBytes:    program.StdoutLogFileMaxBytes,
			RedirectStderr:           program.RedirectStderr,
			StderrLogFile:            program.StderrLogFile,
			StderrLogfileBackups:     program.StderrLogfileBackups,
			StderrLogFileMaxBytes:    program.StderrLogFileMaxBytes,
			DependsOn:                program.DependsOn,
			Labels:                   program.Labels,
		}

		if as := program.AutoRestart; as != nil {
			if *as {
				proc.AutoRestart = AutoStartModeAlways
			} else {
				proc.AutoRestart = AutoStartModeNever
			}
		} else {
			proc.AutoRestart = AutoStartModeDefault
		}

		raw, _ := txn.First("process", "id", program.Name)
		if orig, ok := raw.(*Process); ok && !diff.Changed(orig, proc) {
			continue
		}
		if err := txn.Insert("process", proc); err != nil {
			panic(err)
		}
	}

	deleted := strset.Difference(prev, next)
	if n := deleted.Size(); n > 0 {
		argSlice := make([]interface{}, n)
		for i, e := range deleted.List() {
			argSlice[i] = e
		}
		_, _ = txn.DeleteAll("process", "id", argSlice...)
	}

	return nil
}

func (u *updater) applyHttpServer(txn *memdb.Txn, m *root) {
	if m.HttpServer == nil {
		_ = txn.Delete("server", &Server{Name: "http"})
		return
	}

	server := &Server{
		Name:     "http",
		Address:  m.HttpServer.Port,
		Username: m.HttpServer.Username,
		Password: m.HttpServer.Password,
	}

	raw, _ := txn.First("server", "id", "http")
	if orig, ok := raw.(*Server); ok && !diff.Changed(orig, server) {
		return
	}
	_ = txn.Insert("server", server)
}

func (u *updater) applyGrpcServer(txn *memdb.Txn, m *root) {
	if m.GrpcServer == nil {
		_ = txn.Delete("server", &Server{Name: "grpc"})
		return
	}

	server := &Server{
		Name:     "grpc",
		Address:  m.GrpcServer.Address,
		Username: m.GrpcServer.Username,
		Password: m.GrpcServer.Password,
	}

	raw, _ := txn.First("server", "id", "grpc")
	if orig, ok := raw.(*Server); ok && !diff.Changed(orig, server) {
		return
	}
	_ = txn.Insert("server", server)
}

func (u *updater) applyFileSystem(txn *memdb.Txn, m *root) {
	if m.FileSystem == nil {
		_, _ = txn.DeleteAll("file", "id", "")
		return
	}

	for _, f := range m.FileSystem {
		f := f

		raw, _ := txn.First("file", "id", f.Name)
		if orig, ok := raw.(*file); ok && !diff.Changed(orig, f) {
			return
		}
		_ = txn.Insert("file", &f)
	}
}
