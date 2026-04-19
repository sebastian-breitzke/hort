package vault

import (
	"fmt"
	"os"
	"path/filepath"
)

// LockPath returns the path to the primary vault's lock file.
func LockPath() (string, error) {
	ref, err := PrimaryRef()
	if err != nil {
		return "", err
	}
	return ref.LockPath, nil
}

// lockVault acquires an exclusive file lock on the given vault ref.
func lockVault(ref VaultRef) (func() error, error) {
	if err := os.MkdirAll(filepath.Dir(ref.LockPath), 0700); err != nil {
		return nil, fmt.Errorf("creating lock directory: %w", err)
	}

	file, err := os.OpenFile(ref.LockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening vault lock: %w", err)
	}

	if err := lockFile(file); err != nil {
		_ = file.Close()
		return nil, err
	}

	return func() error {
		return unlockFile(file)
	}, nil
}
