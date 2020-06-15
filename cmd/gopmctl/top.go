package main

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/cobra"
	"github.com/stuartcarnie/gopm/process"
)

var topCmd = cobra.Command{
	Use:   "top",
	Short: "Display the resource usage of a list of processes",
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := control.client.GetProcessInfo(context.Background(), &empty.Empty{})
		if err != nil {
			return err
		}
		display := make(map[string]bool)
		for _, process := range args {
			display[process] = true

		}
		var usages []*processResourceUsage
		for _, p := range res.Processes {
			if !control.inProcessMap(p, display) {
				continue
			}
			info, err := process.Stat(int(p.Pid))
			if err != nil {
				return err
			}
			usages = append(usages, &processResourceUsage{
				ProcessInfo: p,
				Usage:       info,
			})
		}
		control.printTop(usages)
		return nil
	},
}
