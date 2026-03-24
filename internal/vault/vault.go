package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Entry represents a single secret or config entry.
type Entry struct {
	Description  string            `json:"description"`
	Environments map[string]string `json:"environments"`
}

// VaultData is the decrypted vault content.
type VaultData struct {
	Version int              `json:"version"`
	Secrets map[string]Entry `json:"secrets"`
	Configs map[string]Entry `json:"configs"`
}

// NewVaultData creates an empty vault.
func NewVaultData() *VaultData {
	return &VaultData{
		Version: 1,
		Secrets: make(map[string]Entry),
		Configs: make(map[string]Entry),
	}
}

// HortDir returns the hort config directory path.
func HortDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".hort"), nil
}

// VaultPath returns the path to the vault file.
func VaultPath() (string, error) {
	dir, err := HortDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vault.enc"), nil
}

// VaultExists checks if a vault file exists.
func VaultExists() (bool, error) {
	path, err := VaultPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// CreateVault creates a new encrypted vault with the given passphrase.
// Returns the derived key for session caching.
func CreateVault(passphrase []byte) ([]byte, error) {
	dir, err := HortDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating hort directory: %w", err)
	}

	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	params := DefaultArgonParams()
	key := DeriveKey(passphrase, salt, params)

	data := NewVaultData()
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling vault: %w", err)
	}

	encrypted, err := Encrypt(plaintext, passphrase, salt, params)
	if err != nil {
		return nil, err
	}

	path, err := VaultPath()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		return nil, fmt.Errorf("writing vault: %w", err)
	}

	return key, nil
}

// LoadVault reads and decrypts the vault using a session key.
// Returns the vault data and the raw encrypted bytes (needed for re-encryption).
func LoadVault(key []byte) (*VaultData, []byte, error) {
	path, err := VaultPath()
	if err != nil {
		return nil, nil, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading vault: %w", err)
	}

	plaintext, err := DecryptWithKey(raw, key)
	if err != nil {
		return nil, nil, err
	}

	var data VaultData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, nil, fmt.Errorf("parsing vault: %w", err)
	}

	if data.Secrets == nil {
		data.Secrets = make(map[string]Entry)
	}
	if data.Configs == nil {
		data.Configs = make(map[string]Entry)
	}

	return &data, raw, nil
}

// SaveVault encrypts and writes the vault data.
func SaveVault(data *VaultData, key []byte, existingRaw []byte) error {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling vault: %w", err)
	}

	encrypted, err := EncryptWithKey(plaintext, key, existingRaw)
	if err != nil {
		return err
	}

	path, err := VaultPath()
	if err != nil {
		return err
	}

	return os.WriteFile(path, encrypted, 0600)
}

// UnlockVault decrypts the vault with a passphrase and returns the derived key.
func UnlockVault(passphrase []byte) ([]byte, error) {
	path, err := VaultPath()
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading vault: %w", err)
	}

	// Try decrypting to verify passphrase
	_, err = Decrypt(raw, passphrase)
	if err != nil {
		return nil, err
	}

	// Derive the key for session storage
	if len(raw) < HeaderSize {
		return nil, fmt.Errorf("vault file too short")
	}
	salt := raw[0:SaltSize]
	params := ArgonParams{
		Time:    uint32(raw[SaltSize])<<24 | uint32(raw[SaltSize+1])<<16 | uint32(raw[SaltSize+2])<<8 | uint32(raw[SaltSize+3]),
		Memory:  uint32(raw[SaltSize+4])<<24 | uint32(raw[SaltSize+5])<<16 | uint32(raw[SaltSize+6])<<8 | uint32(raw[SaltSize+7]),
		Threads: raw[SaltSize+8],
	}

	key := DeriveKey(passphrase, salt, params)
	return key, nil
}
