package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/stuartcarnie/gopm"
	"github.com/stuartcarnie/gopm/internal/zap/encoder"
	"github.com/stuartcarnie/gopm/pkg/env"
	"github.com/stuartcarnie/gopm/process"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	cfg := zap.NewDevelopmentConfig()
	encoding := "term-color"
	if os.Getenv("NO_COLOR") != "" {
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

func loadEnvFile() error {
	if len(rootOpt.EnvFile) <= 0 {
		return nil
	}

	kvs, err := env.ReadFile(rootOpt.EnvFile)
	if err != nil {
		zap.L().Error("Failed to open environment file", zap.String("file", rootOpt.EnvFile))
		return err
	}
	for i := range kvs {
		kv := &kvs[i]
		err = os.Setenv(kv.Key, kv.Value)
		if err != nil {
			zap.L().Error("Failed to set environment variable", zap.String("key", kv.Key), zap.String("value", kv.Value), zap.Error(err))
		}
	}
	return nil
}

func runServer() error {
	if err := loadEnvFile(); err != nil {
		return err
	}

	s := gopm.NewSupervisor(rootOpt.Configuration)
	if err := s.Reload(); err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

FOR:
	for {
		sig := <-sigs
		zap.L().Info("Received signal to stop all processes and exit", zap.Stringer("signal", sig))
		if rootOpt.QuitDelay == 0 {
			break
		}

		zap.L().Info("Press CTRL-C again to quit", zap.Stringer("signal", sig))
		select {
		case <-sigs:
			break FOR
		case <-time.After(rootOpt.QuitDelay):
			zap.L().Info("Not quitting", zap.Stringer("signal", sig))
		}
	}

	s.GetManager().StopAllProcesses()

	return nil
}

var (
	rootOpt = struct {
		Configuration string
		EnvFile       string
		Shell         string
		QuitDelay     time.Duration
	}{}

	rootCmd = cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			process.SetShellArgs(strings.Split(rootOpt.Shell, " "))
			return runServer()
		},
	}
)

func getDefaultShell() string {
	sh := os.Getenv("SHELL")
	if sh == "" {
		return "/bin/sh -c"
	}

	return sh + " -c"
}

func main() {
	gopm.ReapZombie()

	rootCmd.PersistentFlags().StringVarP(&rootOpt.Configuration, "config", "c", "", "Configuration file")
	flags := rootCmd.Flags()
	flags.StringVar(&rootOpt.EnvFile, "env-file", "", "An optional environment file")
	flags.StringVar(&rootOpt.Shell, "shell", getDefaultShell(), "Specify an alternate shell path")
	flags.DurationVar(&rootOpt.QuitDelay, "quit-delay", time.Second, "Time to wait for second CTRL-C before quitting. 0 to quit immediately.")
	_ = rootCmd.MarkFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to execute command", err)
	}
}
