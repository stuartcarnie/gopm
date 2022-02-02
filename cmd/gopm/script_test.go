package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
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
		"http-server":      httpServerMain,
	}))
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"waitfile":   waitfile,
			"appendfile": appendfile,
			"geturl":     geturl,
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

func httpServerMain() int {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		log.Printf("usage: http-server listenaddr")
		return 2
	}
	err := http.ListenAndServe(args[0], http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(r.URL.Path))
	}))
	if err != nil {
		log.Printf("http.ListenAndServe failed: %v", err)
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

// appendfile appends the contents of the second argument file
// to the first.
// 	appendfile a b
// is like:
//	cat b >> a
func appendfile(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("appendfile doesn't support negation")
	}
	if len(args) != 2 {
		ts.Fatalf("usage: appendfile dstfile appendfile")
	}
	dstFilePath := ts.MkAbs(args[0])
	appendFilePath := ts.MkAbs(args[1])
	dstFile, err := os.OpenFile(dstFilePath, os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		ts.Fatalf("cannot open destination file: %v", err)
	}
	defer dstFile.Close()
	appendFile, err := os.Open(appendFilePath)
	if err != nil {
		ts.Fatalf("cannot open append file: %v", err)
	}
	defer appendFile.Close()
	if _, err := io.Copy(dstFile, appendFile); err != nil {
		ts.Fatalf("error copying data: %v", err)
	}
}

// geturl gets the contents of a url and checks that the status code is 200.
func geturl(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("geturl doesn't support negation")
	}
	if len(args) != 1 {
		ts.Fatalf("usage: geturl URL")
	}
	resp, err := http.Get(args[0])
	if err != nil {
		ts.Fatalf("GET failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		ts.Fatalf("GET returned unexpected response code %v", resp.StatusCode)
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
