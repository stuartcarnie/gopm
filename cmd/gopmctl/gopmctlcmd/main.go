package gopmctlcmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/grpc"

	"github.com/stuartcarnie/gopm/config"
	"github.com/stuartcarnie/gopm/procusage"
	"github.com/stuartcarnie/gopm/rpc"
)

type Control struct {
	Configuration string
	Tags          []string
	Address       string

	client rpc.GopmClient
}

var (
	control = &Control{}

	rootCmd = cobra.Command{
		Use: "gopmctl",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// When all flags have parsed OK, don't show usage info.
			cmd.SilenceUsage = true
			return control.initializeClient()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&control.Configuration, "config", "c", "", "Configuration file")
	rootCmd.PersistentFlags().StringArrayVarP(&control.Tags, "inject", "t", nil, "Set the value of a tagged field in the configuration (for example -t someField=someValue)")
	rootCmd.PersistentFlags().StringVar(&control.Address, "addr", "localhost:9002", "gopm server address")
	rootCmd.AddCommand(&dumpConfigCmd)
	rootCmd.AddCommand(&reloadCmd)
	rootCmd.AddCommand(&restartCmd)
	rootCmd.AddCommand(&shutdownCmd)
	rootCmd.AddCommand(&signalCmd)
	rootCmd.AddCommand(&startAllCmd)
	rootCmd.AddCommand(&startCmd)
	rootCmd.AddCommand(&statusCmd)
	rootCmd.AddCommand(&stopAllCmd)
	rootCmd.AddCommand(&stopCmd)
	rootCmd.AddCommand(&tailLogCmd)
	rootCmd.AddCommand(&topCmd)
}

func Main() int {
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func (ctl *Control) initializeClient() error {
	gc, err := grpc.Dial(ctl.getServerURL(), grpc.WithInsecure())
	if err != nil {
		return err
	}

	control.client = rpc.NewGopmClient(rpc.ClientErrors(gc))
	return nil
}

func (ctl *Control) getServerURL() string {
	if ctl.Address != "" {
		return ctl.Address
	} else if _, err := os.Stat(ctl.Configuration); err == nil {
		cfg, err := config.Load(ctl.Configuration, nil)
		if err == nil {
			// TODO return error from getServerURL
			svr := cfg.GRPCServer
			if svr != nil && svr.Address != "" {
				// TODO network too
				return svr.Address
			}
		}
	}
	return "localhost:9002"
}

// other commands

func (ctl *Control) printProcessInfo(res *rpc.ProcessInfoResponse, processes map[string]bool) {
	tw := tabwriter.NewWriter(os.Stdout, 20, 4, 5, ' ', 0)
	state := func(s string) string {
		return s
	}
	if useColor() {
		state = func(s string) string {
			var av aurora.Value
			switch strings.ToUpper(s) {
			case "RUNNING":
				av = aurora.Green(s)

			case "BACKOFF", "FATAL":
				av = aurora.Red(s)

			default:
				av = aurora.Yellow(s)
			}
			return av.String()
		}
	}
	for _, pinfo := range res.Processes {
		if processes == nil || processes[pinfo.Name] {
			fmt.Fprintf(tw, "%s\t%v\n", pinfo.Name, state(pinfo.State))
		}
	}
	tw.Flush()
}

type processResourceUsage struct {
	*rpc.ProcessInfo
	Usage *procusage.ResourceUsage
}

func (ctl *Control) printTop(processes []*processResourceUsage) {
	tw := tabwriter.NewWriter(os.Stdout, 10, 4, 5, ' ', 0)
	for _, p := range processes {
		fmt.Fprintf(tw, "%s\t%d\t%.1f%%\t%s (%.1f%%)\n",
			p.Name,
			p.Pid,
			p.Usage.CPU,
			p.Usage.HumanResident(),
			p.Usage.Memory,
		)
	}
	tw.Flush()
}

func useColor() bool {
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || !term.IsTerminal(int(os.Stderr.Fd()))
	return !noColor
}
