package gopm

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/process"
	"github.com/stuartcarnie/gopm/rpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	// Version the version of gopm
	Version = "1.0"
)

// Supervisor manage all the processes defined in the supervisor configuration file.
// All the supervisor public interface is defined in this class
type Supervisor struct {
	rpc.UnimplementedGopmServer
	configFile string
	config     *config.Config   // supervisor configuration
	procMgr    *process.Manager // process manager
	httpServer *http.Server
	grpc       *grpc.Server
	restarting bool // if supervisor is in restarting state
}

// NewSupervisor create a Supervisor object with supervisor configuration file
func NewSupervisor(configFile string) *Supervisor {
	return &Supervisor{
		configFile: configFile,
		config:     config.NewConfig(),
		procMgr:    process.NewManager(),
		restarting: false,
	}
}

// GetSupervisorID get the supervisor identifier from configuration file
func (s *Supervisor) GetSupervisorID() string {
	return "supervisor"
}

// IsRestarting check if supervisor is in restarting state
func (s *Supervisor) IsRestarting() bool {
	return s.restarting
}

// Reload reloads the supervisor configuration
func (s *Supervisor) Reload() error {
	changes, err := s.config.LoadPath(s.configFile)
	if err != nil {
		var el Errors
		if errors.As(err, &el) {
			errs := el.Errors()
			zap.L().Error("Error loading configuration")
			for _, err := range errs {
				zap.L().Error("Configuration file error", zap.Error(err))
			}
		} else {
			zap.L().Error("Error loading configuration", zap.Error(err))
		}

		return SupervisorConfigError{Err: err}
	}

	if len(changes) == 0 {
		return nil
	}

	s.createPrograms(changes)
	s.startHTTPServer(changes)
	s.startGrpcServer(changes)
	s.startAutoStartPrograms()

	return nil
}

// WaitForExit wait the supervisor to exit
func (s *Supervisor) WaitForExit() {
	for {
		if s.IsRestarting() {
			s.procMgr.StopAllProcesses()
			break
		}
		time.Sleep(10 * time.Second)
	}
}

func (s *Supervisor) createPrograms(changes memdb.Changes) {
	for _, ch := range changes {
		if ch.Table != "process" {
			continue
		}

		switch {
		case ch.Created(), ch.Updated():
			s.procMgr.CreateOrUpdateProcess(s.GetSupervisorID(), ch.After.(*config.Process))

		case ch.Deleted():
			s.procMgr.RemoveProcess(ch.Before.(*config.Process).Name)
		}
	}
}

func (s *Supervisor) startAutoStartPrograms() {
	s.procMgr.StartAutoStartPrograms()
}

func (s *Supervisor) findServerChange(name string, changes memdb.Changes) *memdb.Change {
	for i := range changes {
		ch := &changes[i]
		if ch.Table != "server" {
			continue
		}

		var id string
		if ch.Deleted() {
			id = ch.Before.(*config.Server).Name
		} else {
			id = ch.After.(*config.Server).Name
		}
		if id == name {
			return ch
		}
	}
	return nil
}

func (s *Supervisor) startHTTPServer(changes memdb.Changes) {
	found := s.findServerChange("http", changes)
	if found == nil {
		return
	}

	var cfg *config.Server
	if found.Updated() || found.Created() {
		cfg = found.After.(*config.Server)
	}

	go func() {
		if s.httpServer != nil {
			err := s.httpServer.Shutdown(context.Background())
			if err != nil {
				zap.L().Error("Unable to shutdown HTTP server", zap.Error(err))
			} else {
				zap.L().Info("Stopped HTTP server")
			}
			s.httpServer = nil
		}

		if cfg == nil {
			return
		}

		grpcServer := grpc.NewServer()
		rpc.RegisterGopmServer(grpcServer, s)
		reflection.Register(grpcServer)
		wrappedGrpc := grpcweb.WrapServer(grpcServer, grpcweb.WithOriginFunc(func(string) bool { return true }))

		mux := http.NewServeMux()
		webguiHandler := NewSupervisorWebgui(s).CreateHandler()
		mux.Handle("/", webguiHandler)

		zap.L().Info("Starting HTTP server", zap.String("addr", cfg.Address))

		srv := http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if wrappedGrpc.IsGrpcWebRequest(req) {
					wrappedGrpc.ServeHTTP(w, req)
					return
				}
				mux.ServeHTTP(w, req)
			}),
			Addr: cfg.Address,
		}
		s.httpServer = &srv
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Error("Unable to start HTTP server", zap.Error(err))
		}
	}()
}

func (s *Supervisor) startGrpcServer(changes memdb.Changes) {
	// restart asynchronously to permit existing Reload request to complete
	found := s.findServerChange("grpc", changes)
	if found == nil {
		return
	}

	var cfg *config.Server
	if found.Updated() || found.Created() {
		cfg = found.After.(*config.Server)
	}

	go func() {
		if s.grpc != nil {
			s.grpc.GracefulStop()
			zap.L().Info("Stopped gRPC server")
			s.grpc = nil
		}

		if cfg == nil {
			return
		}

		ln, err := net.Listen("tcp", cfg.Address)
		if err != nil {
			zap.L().Error("Unable to start gRPC", zap.Error(err), zap.String("addr", cfg.Address))
			return
		}

		grpcServer := grpc.NewServer()
		rpc.RegisterGopmServer(grpcServer, s)
		reflection.Register(grpcServer)
		s.grpc = grpcServer

		zap.L().Info("Starting gRPC server", zap.String("addr", cfg.Address))
		err = grpcServer.Serve(ln)
		if err != nil && err != io.EOF {
			zap.L().Error("Unable to start gRPC server", zap.Error(err))
		}
	}()
}

// GetManager get the Manager object created by superisor
func (s *Supervisor) GetManager() *process.Manager {
	return s.procMgr
}

// SupervisorConfigError is returned when there was a problem loading the
// supervisor configuration file.
type SupervisorConfigError struct {
	Err error
}

func (sc SupervisorConfigError) Error() string {
	return sc.Err.Error()
}

func (sc SupervisorConfigError) Unwrap() error {
	return sc.Err
}
