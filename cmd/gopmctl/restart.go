package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var restartOpt = struct {
	labels map[string]string
}{}

var restartCmd = cobra.Command{
	Use:   "restart",
	Short: "Restart a list of processes",
	RunE: func(cmd *cobra.Command, args []string) error {
		restart := func(name string, labels map[string]string) error {
			req := rpc.StartStopRequest{
				Name:   name,
				Wait:   true,
				Labels: labels,
			}
			_, err := control.client.RestartProcess(context.Background(), &req)
			if status.Code(err) == codes.NotFound {
				fmt.Printf("No processes found: name=%q labels=%s\n", name, labels)
			} else if err != nil {
				return err
			}

			return nil
		}
		if len(args) == 0 {
			if err := restart("", restartOpt.labels); err != nil {
				return err
			}
		}
		for _, name := range args {
			if err := restart(name, restartOpt.labels); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	restartCmd.Flags().StringToStringVarP(&restartOpt.labels, "labels", "l", map[string]string{}, "Labels to apply to")
}
