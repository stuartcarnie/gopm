package gopmctlcmd

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/cobra"
)

var dumpConfigCmd = cobra.Command{
	Use:   "dump-config",
	Short: "Show the configuration as JSON",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := control.client.DumpConfig(context.Background(), &empty.Empty{})
		if err != nil {
			return err
		}
		fmt.Println(res.ConfigJSON)
		return nil
	},
}
