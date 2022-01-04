package gopmctlcmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var signalOpt struct {
	labels map[string]string
}

var signalCmd = cobra.Command{
	Use:   "signal <signal> name [name]...",
	Short: "Send a signal to a list of processes",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sigName := args[0]
		signal := func(name string, labels map[string]string) error {
			req := rpc.SignalProcessRequest{
				Name:   name,
				Signal: sigName,
				Labels: labels,
			}
			_, err := control.client.SignalProcess(context.Background(), &req)
			if status.Code(err) == codes.NotFound {
				fmt.Printf("No processes found: name=%q labels=%s\n", name, labels)
			} else if err != nil {
				return err
			}

			return nil
		}
		if len(args) == 1 {
			if err := signal("", signalOpt.labels); err != nil {
				return err
			}
		}
		for _, name := range args[1:] {
			if err := signal(name, signalOpt.labels); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	signalCmd.Flags().StringToStringVarP(&signalOpt.labels, "labels", "l", map[string]string{}, "Labels to apply to")
}
