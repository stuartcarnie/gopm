package gopmctlcmd

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
)

var tailLogOpt struct {
	backlogLines int
	noFollow     bool
}

func init() {
	tailLogCmd.Flags().IntVarP(&tailLogOpt.backlogLines, "lines", "n", 0, "Number of lines to show from backlog")
	tailLogCmd.Flags().BoolVarP(&tailLogOpt.noFollow, "no-follow", "F", false, "Return immediately at end of log")
}

var tailLogCmd = cobra.Command{
	Use:   "logs",
	Short: "Fetch logs for a process",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return tailLog(context.Background(), args[0])
	},
}

func tailLog(ctx context.Context, name string) error {
	stream, err := control.client.TailLog(ctx, &rpc.TailLogRequest{
		Name:         name,
		NoFollow:     tailLogOpt.noFollow,
		BacklogLines: int64(tailLogOpt.backlogLines),
	})
	if err != nil {
		return err
	}
	var msg rpc.TailLogResponse
	for {
		if err := stream.RecvMsg(&msg); err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
		os.Stdout.Write(msg.Lines)
	}
}
