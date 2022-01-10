package gopmctlcmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/rpc"
)

var startOpt = struct {
	labels map[string]string
}{}

var startCmd = cobra.Command{
	Use:   "start",
	Short: "Start a list of processes",
	RunE: func(cmd *cobra.Command, args []string) error {
		start := func(name string, labels map[string]string) error {
			req := rpc.StartStopRequest{
				Name:   name,
				Wait:   true,
				Labels: labels,
			}
			_, err := control.client.StartProcess(context.Background(), &req)
			var notStartedErr *rpc.NotStartedError
			switch {
			case errors.Is(err, rpc.ErrNotFound):
				fmt.Fprintf(os.Stderr, "No processes found: name=%q labels=%s\n", name, labels)
			case errors.As(err, &notStartedErr):
				fmt.Fprintf(os.Stderr, "Some processes failed to start:\n")
				for _, name := range notStartedErr.ProcessNames {
					fmt.Fprintf(os.Stderr, "\t%s\n", name)
				}
			case err != nil:
				return err
			}
			cmd.SilenceErrors = true
			return err
		}
		if len(args) == 0 {
			if err := start("", startOpt.labels); err != nil {
				return err
			}
		}
		for _, name := range args {
			if err := start(name, startOpt.labels); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	startCmd.Flags().StringToStringVarP(&startOpt.labels, "labels", "l", map[string]string{}, "Labels to apply to")
}
