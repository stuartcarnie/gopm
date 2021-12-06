package gopm

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stuartcarnie/gopm/logger"
	"github.com/stuartcarnie/gopm/process"
	"github.com/stuartcarnie/gopm/rpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ rpc.GopmServer = (*Supervisor)(nil)

func getRpcProcessInfo(proc *process.Process) *rpc.ProcessInfo {
	return &rpc.ProcessInfo{
		Name:        proc.Name(),
		Description: proc.Description(),
		Start:       proc.StartTime().Unix(),
		Stop:        proc.StopTime().Unix(),
		Now:         time.Now().Unix(),
		State:       proc.State().String(),
		SpawnErr:    "",
		ExitStatus:  int64(proc.ExitStatus()),
		Logfile:     proc.Logfile(),
		Pid:         int64(proc.Pid()),
	}
}

func (s *Supervisor) GetProcessInfo(_ context.Context, _ *empty.Empty) (*rpc.ProcessInfoResponse, error) {

	var processes rpc.ProcessInfos
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		processes = append(processes, getRpcProcessInfo(proc))
	})

	processes.SortByName()
	return &rpc.ProcessInfoResponse{Processes: processes}, nil
}

func (s *Supervisor) StartProcess(_ context.Context, req *rpc.StartStopRequest) (*rpc.StartStopResponse, error) {
	procs := s.procMgr.FindMatchWithLabels(req.Name, req.Labels)
	if len(procs) <= 0 {
		return nil, status.Error(codes.NotFound, "No processes found")
	}
	for _, proc := range procs {
		proc.Start(req.Wait)
	}
	return &rpc.StartStopResponse{}, nil
}

func (s *Supervisor) StopProcess(_ context.Context, req *rpc.StartStopRequest) (*rpc.StartStopResponse, error) {
	procs := s.procMgr.FindMatchWithLabels(req.Name, req.Labels)
	if len(procs) <= 0 {
		return nil, status.Error(codes.NotFound, "No processes found")
	}
	for _, proc := range procs {
		proc.Stop(req.Wait)
	}
	return &rpc.StartStopResponse{}, nil
}

func (s *Supervisor) RestartProcess(_ context.Context, req *rpc.StartStopRequest) (*rpc.StartStopResponse, error) {
	procs := s.procMgr.FindMatchWithLabels(req.Name, req.Labels)
	if len(procs) <= 0 {
		return nil, status.Error(codes.NotFound, "No processes found")
	}
	for _, proc := range procs {
		proc.Stop(true)
		proc.Start(req.Wait)
	}
	return &rpc.StartStopResponse{}, nil
}

func (s *Supervisor) StartAllProcesses(_ context.Context, req *rpc.StartStopAllRequest) (*rpc.ProcessInfoResponse, error) {
	var (
		g     errgroup.Group
		procs []*process.Process
	)

	s.procMgr.ForEachProcess(func(p *process.Process) {
		procs = append(procs, p)
		g.Go(func() error {
			p.Start(req.Wait)
			return nil
		})
	})
	_ = g.Wait()

	var res rpc.ProcessInfoResponse
	for _, proc := range procs {
		res.Processes = append(res.Processes, getRpcProcessInfo(proc))
	}

	return &res, nil
}

func (s *Supervisor) StopAllProcesses(_ context.Context, req *rpc.StartStopAllRequest) (*rpc.ProcessInfoResponse, error) {
	var (
		g     errgroup.Group
		procs []*process.Process
	)

	s.procMgr.ForEachProcess(func(p *process.Process) {
		procs = append(procs, p)
		g.Go(func() error {
			p.Stop(req.Wait)
			return nil
		})
	})
	_ = g.Wait()

	var res rpc.ProcessInfoResponse
	for _, proc := range procs {
		res.Processes = append(res.Processes, getRpcProcessInfo(proc))
	}

	return &res, nil
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
	err := s.Reload()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &rpc.ReloadConfigResponse{
		AddedGroup:   nil,
		ChangedGroup: nil,
		RemovedGroup: nil,
	}, nil
}

func (s *Supervisor) TailLog(req *rpc.TailLogRequest, stream rpc.Gopm_TailLogServer) error {
	procMgr := s.GetManager()
	proc := procMgr.Find(req.Name)
	if proc == nil {
		return status.Error(codes.NotFound, "Process not found")
	}

	ctx := stream.Context()
	var res rpc.TailLogResponse
	if req.BacklogLines > 0 {
		_, data := proc.OutputBacklog.Bytes()
		res.Lines = lastNLines(data, int(req.BacklogLines))
		log.Printf("lastNLlines %q -> %q", data, res.Lines)
		if err := stream.Send(&res); err != nil {
			return err
		}
		if req.NoFollow {
			return nil
		}
		// We can miss some data here because there can be a write
		// between the Bytes call above and adding the new logger.
		// TODO fix it so that we can't miss data.
	}

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	plog := logger.NewStdLogger(pw)
	proc.OutputLog.AddLogger(plog)
	go func() {
		<-ctx.Done()
		pr.Close()
	}()

	buf := make([]byte, 8192)
outer:
	for {
		n, err := pr.Read(buf)
		if err != nil {
			break outer
		}
		res.Lines = buf[0:n]
		if err := stream.Send(&res); err != nil {
			break outer
		}
	}
	proc.OutputLog.RemoveLogger(plog)

	return nil
}

func (s *Supervisor) SignalProcess(_ context.Context, req *rpc.SignalProcessRequest) (*empty.Empty, error) {
	procs := s.procMgr.FindMatchWithLabels(req.Name, req.Labels)
	if len(procs) <= 0 {
		return nil, status.Error(codes.NotFound, "No processes found")
	}
	sig, err := req.Signal.ToSignal()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid signal")
	}

	for _, proc := range procs {
		// TODO(sgc): collect errors?
		_ = proc.Signal(sig)
	}

	return &empty.Empty{}, nil
}

func (s *Supervisor) SignalAllProcesses(_ context.Context, req *rpc.SignalProcessRequest) (*rpc.ProcessInfoResponse, error) {
	sig, err := req.Signal.ToSignal()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid signal")
	}

	var res rpc.ProcessInfoResponse
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		proc.Signal(sig)
		res.Processes = append(res.Processes, getRpcProcessInfo(proc))
	})

	return &res, nil
}

// lastNLines walks backwards through a buffer to identify the location of the
// newline preceeding the Nth line of the content.
func lastNLines(b []byte, n int) []byte {
	if len(b) == 0 || n <= 0 {
		return nil
	}
	// We'll count the last line as a full line even if it doesn't
	// end with a newline.
	i := len(b)
	if b[len(b)-1] == '\n' {
		i--
	}
	for nfound := 0; nfound < n; {
		nl := bytes.LastIndexByte(b[:i], '\n')
		if nl == -1 {
			return b
		}
		nfound++
		i = nl
	}
	return b[i+1:]
}
