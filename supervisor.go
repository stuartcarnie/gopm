package gopm

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/process"
	"github.com/stuartcarnie/gopm/rpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Supervisor manage all the processes defined in the supervisor configuration file.
// All the supervisor public interface is defined in this class
type Supervisor struct {
	rpc.UnimplementedGopmServer
	configFile string
	procMgr    *process.Manager // process manager

	// mu guards the fields below it.
	mu         sync.Mutex
	config     *config.Config // supervisor configuration
	fileSystem map[string]*config.File
	httpServer *http.Server
	grpc       *grpc.Server
}

// NewSupervisor create a Supervisor object with supervisor configuration file
func NewSupervisor(configFile string) *Supervisor {
	return &Supervisor{
		configFile: configFile,
		procMgr:    process.NewManager(),
		config:     new(config.Config),
	}
}

// Reload reloads the supervisor configuration
func (s *Supervisor) Reload() error {
	newConfig, err := config.Load(s.configFile, "")
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createFiles(newConfig)
	s.procMgr.Update(context.TODO(), newConfig)
	s.startHTTPServer(newConfig)
	s.startGrpcServer(newConfig)
	s.config = newConfig

	return nil
}

func (s *Supervisor) createFiles(newConfig *config.Config) {
	// Make sure that the root dir always exists even if there are no files in it.
	if err := os.MkdirAll(newConfig.Runtime.Root, 0o777); err != nil {
		zap.L().Error("cannot create root", zap.String("path", newConfig.Runtime.Root), zap.Error(err))
		return
	}
	byPath := make(map[string]*config.File)
	for _, f := range newConfig.FileSystem {
		byPath[filepath.Join(newConfig.Runtime.Root, f.Path)] = f
	}
	for fpath := range s.fileSystem {
		if byPath[fpath] == nil {
			if err := os.Remove(fpath); err != nil {
				zap.L().Error("cannot remove file", zap.String("path", fpath), zap.Error(err))
			}
		}
	}
	for fpath, f := range byPath {
		oldFile := s.fileSystem[fpath]
		if oldFile != nil && oldFile.Content == f.Content {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0777); err != nil {
			zap.L().Error("cannot create directory", zap.Error(err))
			continue
		}
		if err := os.WriteFile(fpath, []byte(f.Content), 0777); err != nil {
			zap.L().Error("cannot create file", zap.Error(err))
			continue
		}
	}
	s.fileSystem = byPath
}

func (s *Supervisor) startHTTPServer(newConfig *config.Config) {
	if reflect.DeepEqual(newConfig.HTTPServer, s.config.HTTPServer) {
		return
	}
	if s.httpServer != nil {
		// TODO is it a problem to do this synchronously?
		err := s.httpServer.Shutdown(context.Background())
		if err != nil {
			zap.L().Error("Unable to shutdown HTTP server", zap.Error(err))
		} else {
			zap.L().Info("Stopped HTTP server")
		}
		s.httpServer = nil
	}
	if newConfig.HTTPServer == nil {
		return
	}

	cfg := newConfig.HTTPServer

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

	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Error("Unable to start HTTP server", zap.Error(err))
		}
	}()
}

func (s *Supervisor) startGrpcServer(newConfig *config.Config) {
	if reflect.DeepEqual(s.config.GRPCServer, newConfig.GRPCServer) {
		return
	}
	var grpcServer *grpc.Server
	var ln net.Listener
	if newConfig.GRPCServer != nil {
		cfg := newConfig.GRPCServer
		netw := cfg.Network
		if netw == "" {
			// TODO default to "tcp" in config
			netw = "tcp"
		}
		// We should be able to listen on the new address
		// before shutting down the old server because
		// we know the address has changed.
		var err error
		ln, err = net.Listen(netw, cfg.Address)
		if err != nil {
			zap.L().Error("Unable to start gRPC", zap.Error(err), zap.String("addr", cfg.Address))
			return
		}
		grpcServer = grpc.NewServer()
		rpc.RegisterGopmServer(grpcServer, s)
		reflection.Register(grpcServer)
	}
	// We can't stop the existing gRPC server gracefully because
	// there's currently an active ReloadConfig call in progress
	// that's waiting for Supervisor.Reload to finish, so
	// if we call GracefulStop synchronously, we'll deadlock.
	// To avoid this, stop the server in the goroutine but update s.grpc
	// immediately.
	grpc := s.grpc
	s.grpc = grpcServer

	go func() {
		if grpc != nil {
			grpc.GracefulStop()
			zap.L().Info("Stopped gRPC server")
		}
		if ln == nil {
			return
		}
		zap.L().Info("Starting gRPC server", zap.Stringer("addr", ln.Addr()))
		if err := grpcServer.Serve(ln); err != nil && err != io.EOF {
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
