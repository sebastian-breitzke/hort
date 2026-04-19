package daemon

import (
	"os"
	"path/filepath"

	"github.com/s16e/hort/internal/vault"
)

// SocketPath returns the Unix-socket path for the daemon. An explicit
// HORT_SOCKET_PATH env var wins — useful for tests on platforms with strict
// socket-path length limits (macOS = ~104 bytes).
func SocketPath() (string, error) {
	if override := os.Getenv("HORT_SOCKET_PATH"); override != "" {
		return override, nil
	}
	dir, err := vault.HortDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.sock"), nil
}

// PIDPath returns the path to the daemon's PID file.
func PIDPath() (string, error) {
	dir, err := vault.HortDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.pid"), nil
}

// LogPath returns the path to the daemon's log file.
func LogPath() (string, error) {
	dir, err := vault.HortDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.log"), nil
}
