package main

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/term"

	"github.com/stuartcarnie/gopm"
	"github.com/stuartcarnie/gopm/internal/zap/encoder"
)

func main() {
	os.Exit(Main())
}

func init() {
	cfg := zap.NewDevelopmentConfig()
	encoding := "term-color"
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || !term.IsTerminal(int(os.Stderr.Fd())) {
		encoding = "term"
	}
	cfg.Encoding = encoding
	cfg.DisableStacktrace = true
	cfg.EncoderConfig = encoder.NewDevelopmentEncoderConfig()
	cfg.EncoderConfig.CallerKey = ""
	cfg.Level.SetLevel(zap.DebugLevel)
	log, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(log)
}

func runServer() error {
	s := gopm.NewSupervisor(rootOpt.Configuration, rootOpt.Tags)
	if err := s.Reload(); err != nil {
		// Don't print configuration errors, as they've already been logged.
		if errors.As(err, &gopm.SupervisorConfigError{}) {
			rootCmd.SilenceErrors = true
		}
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

FOR:
	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGTERM || rootOpt.QuitDelay == 0 {
				break FOR
			}
			zap.L().Info("Received signal to stop all processes and exit")
		case <-s.Done():
			zap.L().Info("Shutdown request received")
			break FOR
		}

		zap.L().Info("Press CTRL-C again to quit")
		select {
		case sig := <-sigs:
			if sig == syscall.SIGTERM || rootOpt.QuitDelay == 0 {
				break FOR
			}
			break FOR
		case <-time.After(rootOpt.QuitDelay):
			zap.L().Info("Not quitting")
		}
	}

	if err := s.Close(); err != nil {
		zap.L().Error("error shutting down", zap.Error(err))
	}
	return nil
}

var (
	rootOpt struct {
		Configuration string
		Tags          []string
		EnvFile       string
		QuitDelay     time.Duration
	}

	rootCmd = cobra.Command{
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// When all flags have parsed OK, don't show usage info.
			cmd.SilenceUsage = true
			return nil
		},
	}
)

// Break initialization loop
func init() {
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runServer()
	}
}

func Main() int {
	rootCmd.PersistentFlags().StringVarP(&rootOpt.Configuration, "config", "c", "", "Configuration directory")
	rootCmd.PersistentFlags().StringArrayVarP(&rootOpt.Tags, "inject", "t", nil, "Set the value of a tagged field in the configuration (for example -t someField=someValue)")
	flags := rootCmd.Flags()
	flags.DurationVar(&rootOpt.QuitDelay, "quit-delay", 2*time.Second, "Time to wait for second CTRL-C before quitting. 0 to quit immediately.")
	_ = rootCmd.MarkFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}
