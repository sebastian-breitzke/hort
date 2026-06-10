package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/s16e/hort/internal/vault"
)

// TokenPath returns the path of the daemon's TCP auth token file.
func TokenPath() (string, error) {
	dir, err := vault.HortDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.token"), nil
}

// EnsureToken reads the TCP auth token, generating a fresh 32-byte hex token
// (mode 0600) on first use. The token gates the daemon's TCP transport;
// clients that cannot reach the Unix socket (e.g. containers) mount or copy
// this file and send the value with every request.
func EnsureToken() (string, error) {
	path, err := TokenPath()
	if err != nil {
		return "", err
	}
	if raw, err := os.ReadFile(path); err == nil {
		token := strings.TrimSpace(string(raw))
		if token != "" {
			return token, nil
		}
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating daemon token: %w", err)
	}
	token := hex.EncodeToString(buf)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0600); err != nil {
		return "", fmt.Errorf("writing daemon token: %w", err)
	}
	return token, nil
}
