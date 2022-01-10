package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/stuartcarnie/gopm/cmd/gopmctl/gopmctlcmd"
	"github.com/stuartcarnie/gopm/config"
)

func TestMain(m *testing.M) {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"gopm":             Main,
		"gopmctl":          gopmctlcmd.Main,
		"interrupt-notify": interruptNotifyMain,
		"signal-notify":    signalNotifyMain,
	}))
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"waitfile": waitfile,
		},
		Setup: func(env *testscript.Env) error {
			env.Setenv("GOTRACEBACK", "all")
			return nil
		},
	})
}

// interruptNotifyMain implements the interrupt-notify command.
// It registers an interrupt signal handler, creates a file
// (signalling to the testscript that it's ready to be stopped),
// waits for an interrupt, then creates another file, signalling
// that the interrupt was successfully handled.
//
// TODO all this polling-based file signalling is a bit retro. We
// could potentially use a unix-domain socket or something, but
// would that really be better overall?
func interruptNotifyMain() int {
	if len(os.Args) != 3 {
		log.Print("usage: interrupt-notify <startfile> <interruptedfile>")
		return 2
	}
	if err := signalNotifyCmd(false, "INT", os.Args[1], os.Args[2]); err != nil {
		log.Printf("interrupt-notify: %v", err)
		return 1
	}
	return 0
}

func signalNotifyMain() int {
	contFlag := flag.Bool("continue", false, "do not stop notifying after first signal")
	flag.Parse()
	args := flag.Args()
	if len(args) != 3 {
		log.Printf("usage: signal-notify [-continue] <signal> <startfile> <interruptedfile> (got %q)", args)
		return 2
	}
	if err := signalNotifyCmd(*contFlag, args[0], args[1], args[2]); err != nil {
		log.Printf("signal-notify: %v", err)
		return 1
	}
	return 0
}

func signalNotifyCmd(cont bool, sigStr string, startFile, interruptedFile string) error {
	sig, err := config.ParseSignal(sigStr)
	if err != nil {
		return err
	}
	nc := make(chan os.Signal, 1)
	signal.Notify(nc, sig.S)
	if err := os.WriteFile(startFile, []byte("started\n"), 0o666); err != nil {
		return err
	}
	for {
		select {
		case <-nc:
			if err := os.WriteFile(interruptedFile, []byte("interrupted\n"), 0o666); err != nil {
				return err
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timed out waiting for %v signal", sig)
		}
		if !cont {
			return nil
		}
	}
}

// waitfile waits until the argument file has been created.
func waitfile(ts *testscript.TestScript, neg bool, args []string) {
	timeout := 5 * time.Second
	if neg {
		// If we're making sure that a file doesn't exist, then
		// a shorter timeout is appropriate.
		timeout = 100 * time.Millisecond
	}
	if len(args) != 1 {
		ts.Fatalf("usage: waitfile file")
	}
	file := ts.MkAbs(args[0])
	deadline := time.Now().Add(timeout)
	found := false
	for time.Now().Before(deadline) {
		_, err := os.Stat(file)
		if err == nil {
			found = true
			break
		}
		if !os.IsNotExist(err) {
			ts.Fatalf("unexpected error waiting for file: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if neg {
		if found {
			ts.Fatalf("file %q unexpectedly found", file)
		}
	} else {
		if !found {
			ts.Fatalf("file %q was not found after %v", file, timeout)
		}
	}
}
