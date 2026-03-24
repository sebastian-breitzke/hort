package store

import (
	"fmt"
	"sort"

	"github.com/s16e/hort/internal/vault"
)

const BaselineEnv = "*"

// EntryInfo is a summary of an entry for listing.
type EntryInfo struct {
	Name         string
	Type         string // "secret" or "config"
	Description  string
	Environments []string
}

// Store provides high-level operations on the vault.
type Store struct {
	key []byte
}

// New creates a store with the given session key.
func New(key []byte) *Store {
	return &Store{key: key}
}

// NewFromSession creates a store using the current session key.
func NewFromSession() (*Store, error) {
	key, err := vault.LoadSession()
	if err != nil {
		return nil, err
	}
	return &Store{key: key}, nil
}

// GetSecret retrieves a secret value. Falls back to baseline (*) if env-specific value doesn't exist.
func (s *Store) GetSecret(name, env string) (string, error) {
	data, _, err := vault.LoadVault(s.key)
	if err != nil {
		return "", err
	}

	entry, ok := data.Secrets[name]
	if !ok {
		return "", fmt.Errorf("secret %q not found", name)
	}

	return resolveEnv(entry, env)
}

// GetConfig retrieves a config value. Falls back to baseline (*) if env-specific value doesn't exist.
func (s *Store) GetConfig(name, env string) (string, error) {
	data, _, err := vault.LoadVault(s.key)
	if err != nil {
		return "", err
	}

	entry, ok := data.Configs[name]
	if !ok {
		return "", fmt.Errorf("config %q not found", name)
	}

	return resolveEnv(entry, env)
}

// SetSecret stores a secret value.
func (s *Store) SetSecret(name, value, env, description string) error {
	if env == "" {
		env = BaselineEnv
	}

	data, raw, err := vault.LoadVault(s.key)
	if err != nil {
		return err
	}

	entry, ok := data.Secrets[name]
	if !ok {
		entry = vault.Entry{
			Environments: make(map[string]string),
		}
	}

	entry.Environments[env] = value
	if description != "" {
		entry.Description = description
	}
	data.Secrets[name] = entry

	return vault.SaveVault(data, s.key, raw)
}

// SetConfig stores a config value.
func (s *Store) SetConfig(name, value, env, description string) error {
	if env == "" {
		env = BaselineEnv
	}

	data, raw, err := vault.LoadVault(s.key)
	if err != nil {
		return err
	}

	entry, ok := data.Configs[name]
	if !ok {
		entry = vault.Entry{
			Environments: make(map[string]string),
		}
	}

	entry.Environments[env] = value
	if description != "" {
		entry.Description = description
	}
	data.Configs[name] = entry

	return vault.SaveVault(data, s.key, raw)
}

// List returns all entries, optionally filtered by type.
func (s *Store) List(typeFilter string) ([]EntryInfo, error) {
	data, _, err := vault.LoadVault(s.key)
	if err != nil {
		return nil, err
	}

	var entries []EntryInfo

	if typeFilter == "" || typeFilter == "secret" {
		for name, entry := range data.Secrets {
			entries = append(entries, EntryInfo{
				Name:         name,
				Type:         "secret",
				Description:  entry.Description,
				Environments: sortedKeys(entry.Environments),
			})
		}
	}

	if typeFilter == "" || typeFilter == "config" {
		for name, entry := range data.Configs {
			entries = append(entries, EntryInfo{
				Name:         name,
				Type:         "config",
				Description:  entry.Description,
				Environments: sortedKeys(entry.Environments),
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type < entries[j].Type
		}
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// Describe returns detailed info about a single entry.
func (s *Store) Describe(name string) (*EntryInfo, error) {
	data, _, err := vault.LoadVault(s.key)
	if err != nil {
		return nil, err
	}

	if entry, ok := data.Secrets[name]; ok {
		return &EntryInfo{
			Name:         name,
			Type:         "secret",
			Description:  entry.Description,
			Environments: sortedKeys(entry.Environments),
		}, nil
	}

	if entry, ok := data.Configs[name]; ok {
		return &EntryInfo{
			Name:         name,
			Type:         "config",
			Description:  entry.Description,
			Environments: sortedKeys(entry.Environments),
		}, nil
	}

	return nil, fmt.Errorf("entry %q not found", name)
}

// Delete removes an entry or a specific environment override.
func (s *Store) Delete(name, env string) error {
	data, raw, err := vault.LoadVault(s.key)
	if err != nil {
		return err
	}

	deleted := false

	if env == "" {
		// Delete entire entry
		if _, ok := data.Secrets[name]; ok {
			delete(data.Secrets, name)
			deleted = true
		}
		if _, ok := data.Configs[name]; ok {
			delete(data.Configs, name)
			deleted = true
		}
	} else {
		// Delete specific environment
		if entry, ok := data.Secrets[name]; ok {
			if _, envOk := entry.Environments[env]; envOk {
				delete(entry.Environments, env)
				data.Secrets[name] = entry
				deleted = true
			}
		}
		if entry, ok := data.Configs[name]; ok {
			if _, envOk := entry.Environments[env]; envOk {
				delete(entry.Environments, env)
				data.Configs[name] = entry
				deleted = true
			}
		}
	}

	if !deleted {
		if env == "" {
			return fmt.Errorf("entry %q not found", name)
		}
		return fmt.Errorf("environment %q not found for %q", env, name)
	}

	return vault.SaveVault(data, s.key, raw)
}

func resolveEnv(entry vault.Entry, env string) (string, error) {
	if env == "" {
		env = BaselineEnv
	}

	if val, ok := entry.Environments[env]; ok {
		return val, nil
	}

	// Fallback to baseline
	if env != BaselineEnv {
		if val, ok := entry.Environments[BaselineEnv]; ok {
			return val, nil
		}
	}

	return "", fmt.Errorf("no value for environment %q (and no baseline)", env)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
