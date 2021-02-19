// +build !windows

package signals

import (
	"fmt"
	"os"
	"syscall"
)

// ToSignal convert a signal name to signal
func ToSignal(signalName string) (os.Signal, error) {
	switch signalName {
	case "HUP":
		return syscall.SIGHUP, nil
	case "INT":
		return syscall.SIGINT, nil
	case "QUIT":
		return syscall.SIGQUIT, nil
	case "KILL":
		return syscall.SIGKILL, nil
	case "USR1":
		return syscall.SIGUSR1, nil
	case "USR2":
		return syscall.SIGUSR2, nil
	case "TERM":
		return syscall.SIGTERM, nil
	case "STOP":
		return syscall.SIGSTOP, nil
	case "CONT":
		return syscall.SIGCONT, nil
	default:
		return syscall.SIGTERM, fmt.Errorf("invalid signal: %s", signalName)
	}
}

// Kill send signal to the process
//
// Args:
//    process - the process which the signal should be sent to
//    sig - the signal will be sent
//    sigChildren - true if the signal needs to be sent to the children also
//
func Kill(process *os.Process, sig os.Signal, sigChildren bool) error {
	localSig := sig.(syscall.Signal)
	pid := process.Pid
	if sigChildren {
		pid = -pid
	}
	return syscall.Kill(pid, localSig)
}
