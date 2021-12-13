package gopm

import (
	"context"
	"errors"
	"fmt"
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
	done       chan struct{}
}

// NewSupervisor create a Supervisor object with supervisor configuration file
func NewSupervisor(configFile string) *Supervisor {
	return &Supervisor{
		configFile: configFile,
		procMgr:    process.NewManager(),
		config:     new(config.Config),
		done:       make(chan struct{}),
	}
}

// Close closes the server, stopping all processes gracefully.
func (s *Supervisor) Close() error {
	// Updating with the empty configuration should gracefully stop all processes
	// in the correct order and stop and tear down any current network
	// requests.
	if err := s.procMgr.Update(context.TODO(), &config.Config{}); err != nil {
		zap.L().Info("cannot terminate processes", zap.Error(err))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.grpc != nil {
		s.grpc.GracefulStop()
		s.grpc = nil
	}
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(context.Background()); err != nil {
			zap.L().Info("cannot terminate HTTP server", zap.Error(err))
		}
		s.httpServer = nil
	}
	return nil
}

// Done returns a channel that's closed when the supervisor gets a shutdown request.
func (s *Supervisor) Done() <-chan struct{} {
	return s.done
}

// Reload reloads the supervisor configuration
func (s *Supervisor) Reload() error {
	zap.L().Info("reloading configuration")
	newConfig, err := config.Load(s.configFile)
	if err != nil {
		zap.L().Error("Error loading configuration", zap.Error(err))
		var configErr *config.ConfigError
		if errors.As(err, &configErr) {
			// TODO zap doesn't seem appropriate for this. Probably
			// better to return all the information in the gRPC response
			// to be printed to the user.
			fmt.Fprintf(os.Stderr, "%s\n", configErr.AllErrors())
		}
		return SupervisorConfigError{Err: err}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Listen on the new server addresses before committing to anything,
	// so that we don't break the network irremediably.
	var httpListener net.Listener
	if newConfig.HTTPServer != nil && !reflect.DeepEqual(newConfig.HTTPServer, s.config.HTTPServer) {
		l, err := net.Listen(newConfig.HTTPServer.Network, newConfig.HTTPServer.Address)
		if err != nil {
			return fmt.Errorf("cannot listen on HTTP server address: %v", err)
		}
		httpListener = l
	}
	var gRPCListener net.Listener
	if newConfig.GRPCServer != nil && !reflect.DeepEqual(newConfig.GRPCServer, s.config.GRPCServer) {
		l, err := net.Listen(newConfig.GRPCServer.Network, newConfig.GRPCServer.Address)
		if err != nil {
			if httpListener != nil {
				httpListener.Close()
			}
			return fmt.Errorf("cannot listen on gRPC server address: %v", err)
		}
		gRPCListener = l
	}

	s.startHTTPServer(newConfig, httpListener)
	s.startGrpcServer(newConfig, gRPCListener)

	s.createFiles(newConfig)
	s.procMgr.Update(context.TODO(), newConfig)
	s.config = newConfig

	return nil
}

func (s *Supervisor) createFiles(newConfig *config.Config) {
	// Make sure that the root dir always exists even if there are no files in it.
	if err := os.MkdirAll(newConfig.Root, 0o777); err != nil {
		zap.L().Error("cannot create root", zap.String("path", newConfig.Root), zap.Error(err))
		return
	}
	byPath := make(map[string]*config.File)
	for _, f := range newConfig.FileSystem {
		byPath[filepath.Join(newConfig.Root, f.Path)] = f
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

func (s *Supervisor) startHTTPServer(newConfig *config.Config, listener net.Listener) {
	if reflect.DeepEqual(newConfig.HTTPServer, s.config.HTTPServer) {
		return
	}
	if listener != nil {
		zap.L().Info("Starting HTTP server", zap.String("addr", "http://"+listener.Addr().String()+"/"))
	} else {
		zap.L().Info("Stopping HTTP server")
	}

	var httpServer *http.Server
	if listener != nil {
		cfg := newConfig.HTTPServer

		grpcServer := grpc.NewServer()
		rpc.RegisterGopmServer(grpcServer, s)
		reflection.Register(grpcServer)
		wrappedGrpc := grpcweb.WrapServer(grpcServer, grpcweb.WithOriginFunc(func(string) bool { return true }))

		mux := http.NewServeMux()
		webguiHandler := NewSupervisorWebgui(s).CreateHandler()
		mux.Handle("/", webguiHandler)

		httpServer = &http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if wrappedGrpc.IsGrpcWebRequest(req) {
					wrappedGrpc.ServeHTTP(w, req)
					return
				}
				mux.ServeHTTP(w, req)
			}),
			Addr: cfg.Address,
		}
	}
	oldServer := s.httpServer
	s.httpServer = httpServer
	go func() {
		// Note: we can't shut down the old server synchronously
		// before starting this goroutine as that might deadlock
		// because it waits for all current requests to
		// complete, which may include a request on the server
		// we're shutting down.
		if oldServer != nil {
			if err := oldServer.Shutdown(context.Background()); err != nil {
				zap.L().Error("Unable to shutdown HTTP server", zap.Error(err))
			} else {
				zap.L().Info("Stopped HTTP server")
			}
		}
		if listener == nil {
			return
		}
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Error("Unable to start HTTP server", zap.Error(err))
		}
	}()
}

func (s *Supervisor) startGrpcServer(newConfig *config.Config, listener net.Listener) {
	if reflect.DeepEqual(s.config.GRPCServer, newConfig.GRPCServer) {
		return
	}
	var grpcServer *grpc.Server
	if newConfig.GRPCServer != nil {
		grpcServer = grpc.NewServer()
		rpc.RegisterGopmServer(grpcServer, s)
		reflection.Register(grpcServer)

	}
	oldServer := s.grpc
	s.grpc = grpcServer

	go func() {
		// Note: we can't shut down the old server synchronously
		// before starting this goroutine as that might deadlock
		// because it waits for all current requests to
		// complete, which may include a request on the server
		// we're shutting down.
		if oldServer != nil {
			oldServer.GracefulStop()
			zap.L().Info("Stopped gRPC server")
		}
		if listener == nil {
			return
		}
		zap.L().Info("Starting gRPC server", zap.Stringer("addr", listener.Addr()))
		if err := grpcServer.Serve(listener); err != nil && err != io.EOF {
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
