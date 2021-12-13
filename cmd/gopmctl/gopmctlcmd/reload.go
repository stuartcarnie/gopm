package gopmctlcmd

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/cobra"
)

var reloadCmd = cobra.Command{
	Use:   "reload",
	Short: "Reload the configuration for the gopm process",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := control.client.ReloadConfig(context.Background(), &empty.Empty{})
		return err
	},
}
