package store

import (
	"fmt"
	"sort"
	"strings"

	"github.com/s16e/hort/internal/vault"
)

// EntryInfo is a summary of an entry for listing.
type EntryInfo struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Description  string   `json:"description"`
	Environments []string `json:"environments"`
	Contexts     []string `json:"contexts"`
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

// GetSecret retrieves a secret value with env+context fallback.
// Fallback: env+ctx → env+* → *+*
func (s *Store) GetSecret(name, env, context string) (string, error) {
	data, _, err := vault.LoadVault(s.key)
	if err != nil {
		return "", err
	}

	entry, ok := data.Secrets[name]
	if !ok {
		return "", fmt.Errorf("secret %q not found", name)
	}

	return resolve(entry, env, context)
}

// GetConfig retrieves a config value with env+context fallback.
func (s *Store) GetConfig(name, env, context string) (string, error) {
	data, _, err := vault.LoadVault(s.key)
	if err != nil {
		return "", err
	}

	entry, ok := data.Configs[name]
	if !ok {
		return "", fmt.Errorf("config %q not found", name)
	}

	return resolve(entry, env, context)
}

// SetSecret stores a secret value.
func (s *Store) SetSecret(name, value, env, context, description string) error {
	return vault.UpdateVault(s.key, func(data *vault.VaultData) error {
		entry, ok := data.Secrets[name]
		if !ok {
			entry = vault.Entry{
				Values: make(map[string]string),
			}
		}

		key := vault.MakeLookupKey(env, context)
		entry.Values[key] = value
		if description != "" {
			entry.Description = description
		}
		data.Secrets[name] = entry
		return nil
	})
}

// SetConfig stores a config value.
func (s *Store) SetConfig(name, value, env, context, description string) error {
	return vault.UpdateVault(s.key, func(data *vault.VaultData) error {
		entry, ok := data.Configs[name]
		if !ok {
			entry = vault.Entry{
				Values: make(map[string]string),
			}
		}

		key := vault.MakeLookupKey(env, context)
		entry.Values[key] = value
		if description != "" {
			entry.Description = description
		}
		data.Configs[name] = entry
		return nil
	})
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
			entries = append(entries, entryToInfo(name, "secret", entry))
		}
	}

	if typeFilter == "" || typeFilter == "config" {
		for name, entry := range data.Configs {
			entries = append(entries, entryToInfo(name, "config", entry))
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
		info := entryToInfo(name, "secret", entry)
		return &info, nil
	}

	if entry, ok := data.Configs[name]; ok {
		info := entryToInfo(name, "config", entry)
		return &info, nil
	}

	return nil, fmt.Errorf("entry %q not found", name)
}

// Delete removes an entry, a specific env+context combination, or all values for an env.
func (s *Store) Delete(name, env, context string) error {
	return vault.UpdateVault(s.key, func(data *vault.VaultData) error {
		deleted := false

		if env == "" && context == "" {
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
			key := vault.MakeLookupKey(env, context)
			if entry, ok := data.Secrets[name]; ok {
				if _, exists := entry.Values[key]; exists {
					delete(entry.Values, key)
					data.Secrets[name] = entry
					deleted = true
				}
			}
			if entry, ok := data.Configs[name]; ok {
				if _, exists := entry.Values[key]; exists {
					delete(entry.Values, key)
					data.Configs[name] = entry
					deleted = true
				}
			}
		}

		if !deleted {
			return fmt.Errorf("entry %q not found (env=%q, context=%q)", name, env, context)
		}

		return nil
	})
}

// resolve implements the fallback chain: env+ctx → env+* → *+*
func resolve(entry vault.Entry, env, context string) (string, error) {
	if env == "" {
		env = "*"
	}
	if context == "" {
		context = "*"
	}

	// Try exact match: env+context
	if val, ok := entry.Values[vault.MakeLookupKey(env, context)]; ok {
		return val, nil
	}

	// Fallback: env+* (same env, no context)
	if context != "*" {
		if val, ok := entry.Values[vault.MakeLookupKey(env, "*")]; ok {
			return val, nil
		}
	}

	// Fallback: *+* (baseline)
	if env != "*" {
		if val, ok := entry.Values[vault.MakeLookupKey("*", "*")]; ok {
			return val, nil
		}
	}

	return "", fmt.Errorf("no value found (env=%q, context=%q, and no baseline)", env, context)
}

func entryToInfo(name, entryType string, entry vault.Entry) EntryInfo {
	envSet := make(map[string]bool)
	ctxSet := make(map[string]bool)

	for key := range entry.Values {
		env, ctx := vault.ParseLookupKey(key)
		envSet[env] = true
		if ctx != "*" {
			ctxSet[ctx] = true
		}
	}

	return EntryInfo{
		Name:         name,
		Type:         entryType,
		Description:  entry.Description,
		Environments: sortedSetKeys(envSet),
		Contexts:     sortedSetKeys(ctxSet),
	}
}

func sortedSetKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ContextValues returns all context-specific values for an entry+env combination.
// Used when --context is not specified and the agent wants to see all contexts.
func (s *Store) ContextValues(name, entryType, env string) (map[string]string, error) {
	data, _, err := vault.LoadVault(s.key)
	if err != nil {
		return nil, err
	}

	var entries map[string]vault.Entry
	switch entryType {
	case "secret":
		entries = data.Secrets
	case "config":
		entries = data.Configs
	default:
		// Try both
		entries = data.Secrets
		if _, ok := entries[name]; !ok {
			entries = data.Configs
		}
	}

	entry, ok := entries[name]
	if !ok {
		return nil, fmt.Errorf("entry %q not found", name)
	}

	if env == "" {
		env = "*"
	}

	result := make(map[string]string)
	prefix := env + ":"
	for key, val := range entry.Values {
		if strings.HasPrefix(key, prefix) {
			_, ctx := vault.ParseLookupKey(key)
			result[ctx] = val
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no values for %q in env %q", name, env)
	}

	return result, nil
}
