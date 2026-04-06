//go:build !windows

package vault

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func lockFile(file *os.File) error {
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		return fmt.Errorf("locking vault: %w", err)
	}
	return nil
}

func unlockFile(file *os.File) error {
	unlockErr := unix.Flock(int(file.Fd()), unix.LOCK_UN)
	closeErr := file.Close()
	if unlockErr != nil {
		return fmt.Errorf("unlocking vault: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing vault lock: %w", closeErr)
	}
	return nil
}
