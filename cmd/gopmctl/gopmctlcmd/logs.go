package gopmctlcmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
)

type deviceType int

const (
	DeviceTypeStdout deviceType = 1 << iota
	DeviceTypeStderr
	DeviceTypeAll = DeviceTypeStdout | DeviceTypeStderr
)

func (d deviceType) String() string {
	switch d {
	case DeviceTypeStdout:
		return "stdout"
	case DeviceTypeStderr:
		return "stderr"
	case DeviceTypeAll:
		return "all"
	default:
		return "<invalid>"
	}
}

func (d *deviceType) Set(s string) error {
	switch s {
	case "stdout":
		*d = DeviceTypeStdout
	case "stderr":
		*d = DeviceTypeStderr
	case "all":
		*d = DeviceTypeAll
	default:
		return fmt.Errorf("invalid device type: %s", s)
	}

	return nil
}

func (d deviceType) Type() string {
	return "DEVICE-TYPE"
}

var tailLogOpt = struct {
	backlogLines int
	device       deviceType
	noFollow     bool
}{
	device: DeviceTypeAll,
}

var linesPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 4096)
	},
}

func tailLog(ctx context.Context, name string, device rpc.LogDevice) (<-chan []byte, error) {
	req := rpc.TailLogRequest{
		Name:         name,
		Device:       device,
		NoFollow:     tailLogOpt.noFollow,
		BacklogLines: int64(tailLogOpt.backlogLines),
	}

	stream, err := control.client.TailLog(ctx, &req)
	if err != nil {
		return nil, err
	}

	lines := make(chan []byte)

	go func() {
		defer close(lines)

		msg := new(rpc.TailLogResponse)
		for {
			err := stream.RecvMsg(msg)
			if err != nil {
				return
			}

			dst := linesPool.Get().([]byte)
			dst = append(dst[:0], msg.Lines...)
			msg.Lines = msg.Lines[:0]

			select {
			case <-ctx.Done():
				return
			case lines <- dst:
			}
		}
	}()

	return lines, nil
}

var tailLogCmd = cobra.Command{
	Use:   "logs",
	Short: "Fetch logs for a process",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		ctx, cancelFn := signal.NotifyContext(ctx, os.Interrupt)
		defer cancelFn()

		var sources []<-chan []byte

		if tailLogOpt.device&DeviceTypeStdout == DeviceTypeStdout {
			if src, err := tailLog(ctx, args[0], rpc.LogDevice_STDOUT); err != nil {
				return err
			} else {
				sources = append(sources, src)
			}
		}

		if tailLogOpt.device&DeviceTypeStderr == DeviceTypeStderr {
			if src, err := tailLog(ctx, args[0], rpc.LogDevice_STDERR); err != nil {
				return err
			} else {
				sources = append(sources, src)
			}
		}

		for data := range merge(sources...) {
			_, _ = os.Stdout.Write(data)
			linesPool.Put(data)
		}

		return nil
	},
}

func init() {
	tailLogCmd.Flags().VarP(&tailLogOpt.device, "device", "d", "Device to tail (stderr|stdout|all)")
	tailLogCmd.Flags().IntVarP(&tailLogOpt.backlogLines, "lines", "n", 0, "Number of lines to show from backlog")
	tailLogCmd.Flags().BoolVarP(&tailLogOpt.noFollow, "no-follow", "F", false, "Return immediately at end of log")
}
