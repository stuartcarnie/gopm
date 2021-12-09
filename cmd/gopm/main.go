package main

import (
	"errors"
	"fmt"
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
	log, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(log)
}

func runServer() error {
	s := gopm.NewSupervisor(rootOpt.Configuration)
	if err := s.Reload(); err != nil {
		// Ignore config loading errors, because the Supervisor logs those.
		if errors.As(err, &gopm.SupervisorConfigError{}) {
			return nil
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
	rootOpt = struct {
		Configuration string
		EnvFile       string
		QuitDelay     time.Duration
	}{}

	rootCmd = cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer()
		},
	}
)

func Main() int {
	gopm.ReapZombie()

	rootCmd.PersistentFlags().StringVarP(&rootOpt.Configuration, "config", "c", "", "Configuration file")
	flags := rootCmd.Flags()
	flags.StringVar(&rootOpt.EnvFile, "env-file", "", "An optional environment file")
	flags.DurationVar(&rootOpt.QuitDelay, "quit-delay", 2*time.Second, "Time to wait for second CTRL-C before quitting. 0 to quit immediately.")
	_ = rootCmd.MarkFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to execute command", err)
		return 1
	}
	return 0
}
