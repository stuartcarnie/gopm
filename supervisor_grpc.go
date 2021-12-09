package gopm

import (
	"context"
	"errors"
	"os"
	"sort"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stuartcarnie/gopm/process"
	"github.com/stuartcarnie/gopm/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ rpc.GopmServer = (*Supervisor)(nil)

func (s *Supervisor) GetProcessInfo(_ context.Context, _ *empty.Empty) (*rpc.ProcessInfoResponse, error) {
	return s.allProcesses(), nil
}

func mkError(err error) error {
	if errors.Is(err, process.ErrNotFound) {
		return status.Error(codes.NotFound, "No processes found")
	}
	return err
}

func (s *Supervisor) StartProcess(_ context.Context, req *rpc.StartStopRequest) (*rpc.StartStopResponse, error) {
	if err := s.procMgr.StartProcesses(req.Name, req.Labels); err != nil {
		return nil, mkError(err)
	}
	return &rpc.StartStopResponse{}, nil
}

func (s *Supervisor) StopProcess(_ context.Context, req *rpc.StartStopRequest) (*rpc.StartStopResponse, error) {
	if err := s.procMgr.StopProcesses(req.Name, req.Labels); err != nil {
		return nil, mkError(err)
	}
	return &rpc.StartStopResponse{}, nil
}

func (s *Supervisor) RestartProcess(_ context.Context, req *rpc.StartStopRequest) (*rpc.StartStopResponse, error) {
	if err := s.procMgr.RestartProcesses(req.Name, req.Labels); err != nil {
		return nil, mkError(err)
	}
	return &rpc.StartStopResponse{}, nil
}

func (s *Supervisor) StartAllProcesses(_ context.Context, req *rpc.StartStopAllRequest) (*rpc.ProcessInfoResponse, error) {
	s.procMgr.StartAllProcesses()
	return s.allProcesses(), nil
}

func (s *Supervisor) StopAllProcesses(_ context.Context, req *rpc.StartStopAllRequest) (*rpc.ProcessInfoResponse, error) {
	s.procMgr.StopAllProcesses()
	return s.allProcesses(), nil
}

func (s *Supervisor) Shutdown(context.Context, *empty.Empty) (*empty.Empty, error) {
	// TODO(sgc): This is not right
	s.procMgr.StopAllProcesses()
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	return &empty.Empty{}, nil
}

func (s *Supervisor) ReloadConfig(context.Context, *empty.Empty) (*rpc.ReloadConfigResponse, error) {
	if err := s.Reload(); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &rpc.ReloadConfigResponse{
		AddedGroup:   nil,
		ChangedGroup: nil,
		RemovedGroup: nil,
	}, nil
}

func (s *Supervisor) TailLog(req *rpc.TailLogRequest, stream rpc.Gopm_TailLogServer) error {
	err := s.procMgr.TailLog(stream.Context(), process.TailLogParams{
		Name:         req.Name,
		BacklogLines: req.BacklogLines,
		Follow:       !req.NoFollow,
		Writer:       &streamLogger{stream},
	})
	return mkError(err)
}

type streamLogger struct {
	stream rpc.Gopm_TailLogServer
}

func (l *streamLogger) Write(buf []byte) (int, error) {
	if err := l.stream.Send(&rpc.TailLogResponse{
		Lines: buf,
	}); err != nil {
		return 0, err
	}
	return len(buf), nil
}

func (s *Supervisor) SignalProcess(_ context.Context, req *rpc.SignalProcessRequest) (*empty.Empty, error) {
	sig, err := req.Signal.ToSignal()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid signal")
	}
	if err := s.procMgr.SignalProcesses(req.Name, req.Labels, sig); err != nil {
		return nil, mkError(err)
	}
	return &empty.Empty{}, nil
}

func (s *Supervisor) SignalAllProcesses(_ context.Context, req *rpc.SignalProcessRequest) (*rpc.ProcessInfoResponse, error) {
	sig, err := req.Signal.ToSignal()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid signal")
	}
	if err := s.procMgr.SignalProcesses("", nil, sig); err != nil {
		return nil, mkError(err)
	}
	return s.allProcesses(), nil
}

func (s *Supervisor) allProcesses() *rpc.ProcessInfoResponse {
	infos := s.procMgr.AllProcessInfo()
	rpcInfos := make([]*rpc.ProcessInfo, len(infos))
	for i, info := range infos {
		rpcInfos[i] = &rpc.ProcessInfo{
			Name:       info.Name,
			Start:      info.Start.Unix(),
			Stop:       info.Stop.Unix(),
			Now:        time.Now().Unix(),
			State:      info.State.String(),
			ExitStatus: int64(info.ExitStatus),
			Logfile:    info.Logfile,
			Pid:        int64(info.Pid),
		}
	}
	sort.Slice(rpcInfos, func(i, j int) bool {
		return rpcInfos[i].Name < rpcInfos[j].Name
	})
	return &rpc.ProcessInfoResponse{
		Processes: rpcInfos,
	}
}
