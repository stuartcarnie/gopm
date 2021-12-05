package gopmctlcmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var stopOpt = struct {
	labels map[string]string
}{}

var stopCmd = cobra.Command{
	Use:   "stop process...",
	Short: "Stop a list of processes",
	RunE: func(cmd *cobra.Command, args []string) error {
		stop := func(name string, labels map[string]string) error {
			req := rpc.StartStopRequest{
				Name:   name,
				Wait:   true,
				Labels: labels,
			}
			_, err := control.client.StopProcess(context.Background(), &req)
			if status.Code(err) == codes.NotFound {
				fmt.Printf("No processes found: name=%q labels=%s\n", name, labels)
			} else if err != nil {
				return err
			}

			return nil
		}
		if len(args) == 0 {
			if err := stop("", stopOpt.labels); err != nil {
				return err
			}
		}
		for _, name := range args {
			if err := stop(name, stopOpt.labels); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	stopCmd.Flags().StringToStringVarP(&stopOpt.labels, "labels", "l", map[string]string{}, "Labels to apply to")
}
