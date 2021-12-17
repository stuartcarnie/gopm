//go:build windows
// +build windows

package signals

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"go.uber.org/zap"
)

// convert a signal name to signal
func ToSignal(signalName string) (os.Signal, error) {
	if signalName == "HUP" {
		return syscall.SIGHUP, nil
	} else if signalName == "INT" {
		return syscall.SIGINT, nil
	} else if signalName == "QUIT" {
		return syscall.SIGQUIT, nil
	} else if signalName == "KILL" {
		return syscall.SIGKILL, nil
	} else if signalName == "USR1" {
		zap.L().Warn("Signal USR1 is not supported on Windows")
		return nil, errors.New("signal USR1 is not supported on windows")
	} else if signalName == "USR2" {
		zap.L().Warn("Signal USR2 is not supported on Windows")
		return nil, errors.New("signal USR2 is not supported on windows")
	} else {
		return syscall.SIGTERM, nil
	}
}

//
// Args:
//    process - the process
//    sig - the signal
//    sigChildren - ignore in windows system
//
func Kill(process *os.Process, sig os.Signal, sigChilren bool) error {
	// Signal command can't kill children processes, call  taskkill command to kill them
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", process.Pid))
	err := cmd.Start()
	if err == nil {
		return cmd.Wait()
	}
	// if fail to find taskkill, fallback to normal signal
	return process.Signal(sig)
}
