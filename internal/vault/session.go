package vault

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// SessionPath returns the path to the session key file.
func SessionPath() (string, error) {
	dir, err := HortDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".session"), nil
}

// SaveSession stores the derived key in the session file.
func SaveSession(key []byte) error {
	path, err := SessionPath()
	if err != nil {
		return err
	}

	dir, err := HortDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating hort directory: %w", err)
	}

	encoded := hex.EncodeToString(key)
	return os.WriteFile(path, []byte(encoded), 0600)
}

// LoadSession reads the session key. Returns an error if the vault is locked.
func LoadSession() ([]byte, error) {
	path, err := SessionPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("vault is locked — run `hort unlock` first")
	}
	if err != nil {
		return nil, fmt.Errorf("reading session: %w", err)
	}

	key, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("invalid session file: %w", err)
	}

	return key, nil
}

// ClearSession removes the session key file.
func ClearSession() error {
	path, err := SessionPath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil // already locked
	}
	return err
}

// IsUnlocked checks if a session exists.
func IsUnlocked() bool {
	path, err := SessionPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
