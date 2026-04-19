//go:build windows

package cli

import (
	"os"
)

func shutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}

func signalDaemonStop(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
