//go:build windows

package vault

import (
	"fmt"
	"math"
	"os"

	"golang.org/x/sys/windows"
)

func lockFile(file *os.File) error {
	handle := windows.Handle(file.Fd())
	overlapped := new(windows.Overlapped)
	if err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, math.MaxUint32, math.MaxUint32, overlapped); err != nil {
		return fmt.Errorf("locking vault: %w", err)
	}
	return nil
}

func unlockFile(file *os.File) error {
	handle := windows.Handle(file.Fd())
	overlapped := new(windows.Overlapped)
	unlockErr := windows.UnlockFileEx(handle, 0, math.MaxUint32, math.MaxUint32, overlapped)
	closeErr := file.Close()
	if unlockErr != nil {
		return fmt.Errorf("unlocking vault: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing vault lock: %w", closeErr)
	}
	return nil
}
