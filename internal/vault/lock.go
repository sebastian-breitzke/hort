package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// LockPath returns the path to the vault lock file.
func LockPath() (string, error) {
	dir, err := HortDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vault.lock"), nil
}

func lockVault() (func() error, error) {
	dir, err := HortDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating hort directory: %w", err)
	}

	path, err := LockPath()
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening vault lock: %w", err)
	}

	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("locking vault: %w", err)
	}

	return func() error {
		unlockErr := unix.Flock(int(file.Fd()), unix.LOCK_UN)
		closeErr := file.Close()
		if unlockErr != nil {
			return fmt.Errorf("unlocking vault: %w", unlockErr)
		}
		if closeErr != nil {
			return fmt.Errorf("closing vault lock: %w", closeErr)
		}
		return nil
	}, nil
}
