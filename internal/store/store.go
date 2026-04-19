package store

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/s16e/hort/internal/vault"
)

// EntryInfo is a summary of an entry for listing.
type EntryInfo struct {
	Name         string   `json:"name"`
	Source       string   `json:"source"`
	Type         string   `json:"type"`
	Description  string   `json:"description"`
	Environments []string `json:"environments"`
	Contexts     []string `json:"contexts"`
}

// Source is an unlocked vault mounted into the active store.
type Source struct {
	Name       string
	Ref        vault.VaultRef
	SessionKey []byte
	Primary    bool
}

// Store provides high-level operations across the primary vault and any
// unlocked mounted sources.
type Store struct {
	primary *Source
	mounts  []*Source
}

// New creates a store over the given primary + mount sources.
func New(primary *Source, mounts []*Source) *Store {
	return &Store{primary: primary, mounts: mounts}
}

// NewFromSession builds a store from on-disk session files. The primary vault
// must be unlocked; mounted sources that are locked are skipped with a stderr
// warning so merged reads stay best-effort.
func NewFromSession() (*Store, error) {
	primaryRef, err := vault.PrimaryRef()
	if err != nil {
		return nil, err
	}
	primaryKey, err := vault.LoadSessionFor(primaryRef)
	if err != nil {
		return nil, err
	}
	primary := &Source{
		Name:       primaryRef.Name,
		Ref:        primaryRef,
		SessionKey: primaryKey,
		Primary:    true,
	}

	records, err := vault.ListSources()
	if err != nil {
		return nil, err
	}

	var mounts []*Source
	for _, rec := range records {
		ref, err := vault.RefFromRecord(rec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: mount %s has invalid registry entry: %v\n", rec.Name, err)
			continue
		}
		key, err := vault.LoadSessionFor(ref)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: mount %s is locked, skipping merged reads\n", rec.Name)
			continue
		}
		mounts = append(mounts, &Source{
			Name:       rec.Name,
			Ref:        ref,
			SessionKey: key,
		})
	}

	return &Store{primary: primary, mounts: mounts}, nil
}

// Sources returns every source in the store (primary first).
func (s *Store) Sources() []*Source {
	out := make([]*Source, 0, 1+len(s.mounts))
	if s.primary != nil {
		out = append(out, s.primary)
	}
	out = append(out, s.mounts...)
	return out
}

func (s *Store) sourceByName(name string) (*Source, error) {
	if name == "" || name == vault.PrimarySourceName {
		if s.primary == nil {
			return nil, fmt.Errorf("primary vault is not available")
		}
		return s.primary, nil
	}
	for _, src := range s.mounts {
		if src.Name == name {
			return src, nil
		}
	}
	return nil, fmt.Errorf("source %q is not mounted or not unlocked", name)
}

type resolvedHit struct {
	Source string
	Value  string
}

func collectHits(src *Source, entries map[string]vault.Entry, name, env, context string) (resolvedHit, bool) {
	entry, ok := entries[name]
	if !ok {
		return resolvedHit{}, false
	}
	val, ok := resolve(entry, env, context)
	if !ok {
		return resolvedHit{}, false
	}
	return resolvedHit{Source: src.Name, Value: val}, true
}

// GetSecret retrieves a secret value with env+context fallback.
// Merges across all unlocked sources. If more than one source has a value,
// returns a disambiguation error listing the candidates.
func (s *Store) GetSecret(name, env, context string) (string, string, error) {
	return s.getMerged(name, env, context, secretsOf)
}

// GetConfig behaves like GetSecret but for config entries.
func (s *Store) GetConfig(name, env, context string) (string, string, error) {
	return s.getMerged(name, env, context, configsOf)
}

// GetFrom retrieves a value from the named source (primary if empty).
func (s *Store) GetFrom(source, name, env, context, entryType string) (string, error) {
	src, err := s.sourceByName(source)
	if err != nil {
		return "", err
	}
	data, _, err := vault.LoadVault(src.Ref, src.SessionKey)
	if err != nil {
		return "", err
	}
	var entries map[string]vault.Entry
	switch entryType {
	case "secret":
		entries = data.Secrets
	case "config":
		entries = data.Configs
	default:
		return "", fmt.Errorf("unknown entry type %q", entryType)
	}
	hit, ok := collectHits(src, entries, name, env, context)
	if !ok {
		return "", fmt.Errorf("%s %q not found in source %q (env=%q, context=%q)", entryType, name, src.Name, env, context)
	}
	return hit.Value, nil
}

func (s *Store) getMerged(name, env, context string, selector func(*vault.VaultData) map[string]vault.Entry) (string, string, error) {
	var hits []resolvedHit
	for _, src := range s.Sources() {
		data, _, err := vault.LoadVault(src.Ref, src.SessionKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: source %s unreadable: %v\n", src.Name, err)
			continue
		}
		if hit, ok := collectHits(src, selector(data), name, env, context); ok {
			hits = append(hits, hit)
		}
	}
	switch len(hits) {
	case 0:
		return "", "", fmt.Errorf("%q not found (env=%q, context=%q)", name, env, context)
	case 1:
		return hits[0].Value, hits[0].Source, nil
	default:
		names := make([]string, len(hits))
		for i, h := range hits {
			names[i] = h.Source
		}
		sort.Strings(names)
		return "", "", fmt.Errorf("%q is ambiguous across sources [%s] — rerun with --source <name>", name, strings.Join(names, ", "))
	}
}

// SetSecret stores a secret in the target source (primary if source is empty).
func (s *Store) SetSecret(source, name, value, env, context, description string) error {
	return s.set(source, name, value, env, context, description, "secret")
}

// SetConfig stores a config in the target source.
func (s *Store) SetConfig(source, name, value, env, context, description string) error {
	return s.set(source, name, value, env, context, description, "config")
}

func (s *Store) set(source, name, value, env, context, description, entryType string) error {
	src, err := s.sourceByName(source)
	if err != nil {
		return err
	}
	return vault.UpdateVault(src.Ref, src.SessionKey, func(data *vault.VaultData) error {
		var entries map[string]vault.Entry
		switch entryType {
		case "secret":
			entries = data.Secrets
		case "config":
			entries = data.Configs
		default:
			return fmt.Errorf("unknown entry type %q", entryType)
		}
		entry, ok := entries[name]
		if !ok {
			entry = vault.Entry{Values: make(map[string]string)}
		}
		if entry.Values == nil {
			entry.Values = make(map[string]string)
		}
		key := vault.MakeLookupKey(env, context)
		entry.Values[key] = value
		if description != "" {
			entry.Description = description
		}
		entries[name] = entry
		return nil
	})
}

// List returns all entries across all sources, optionally filtered by type.
func (s *Store) List(typeFilter string) ([]EntryInfo, error) {
	var entries []EntryInfo
	for _, src := range s.Sources() {
		data, _, err := vault.LoadVault(src.Ref, src.SessionKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: source %s unreadable: %v\n", src.Name, err)
			continue
		}
		if typeFilter == "" || typeFilter == "secret" {
			for name, entry := range data.Secrets {
				entries = append(entries, entryToInfo(name, src.Name, "secret", entry))
			}
		}
		if typeFilter == "" || typeFilter == "config" {
			for name, entry := range data.Configs {
				entries = append(entries, entryToInfo(name, src.Name, "config", entry))
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Source != entries[j].Source {
			return sourceSortKey(entries[i].Source) < sourceSortKey(entries[j].Source)
		}
		if entries[i].Type != entries[j].Type {
			return entries[i].Type < entries[j].Type
		}
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// Describe returns all matches for a name across sources (one record per source+type hit).
func (s *Store) Describe(name string) ([]EntryInfo, error) {
	var matches []EntryInfo
	for _, src := range s.Sources() {
		data, _, err := vault.LoadVault(src.Ref, src.SessionKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: source %s unreadable: %v\n", src.Name, err)
			continue
		}
		if entry, ok := data.Secrets[name]; ok {
			matches = append(matches, entryToInfo(name, src.Name, "secret", entry))
		}
		if entry, ok := data.Configs[name]; ok {
			matches = append(matches, entryToInfo(name, src.Name, "config", entry))
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("entry %q not found", name)
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Source != matches[j].Source {
			return sourceSortKey(matches[i].Source) < sourceSortKey(matches[j].Source)
		}
		return matches[i].Type < matches[j].Type
	})
	return matches, nil
}

// Delete removes an entry in the given source. An empty env+context deletes
// the entire entry (all its env/context overrides) inside that source only.
func (s *Store) Delete(source, name, env, context string) error {
	src, err := s.sourceByName(source)
	if err != nil {
		return err
	}
	return vault.UpdateVault(src.Ref, src.SessionKey, func(data *vault.VaultData) error {
		deleted := false

		if env == "" && context == "" {
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
			return fmt.Errorf("entry %q not found in source %q (env=%q, context=%q)", name, src.Name, env, context)
		}
		return nil
	})
}

// ContextValues returns all context-specific values for an entry+env combination.
// Searches the given source (primary if empty).
func (s *Store) ContextValues(source, name, entryType, env string) (map[string]string, error) {
	src, err := s.sourceByName(source)
	if err != nil {
		return nil, err
	}
	data, _, err := vault.LoadVault(src.Ref, src.SessionKey)
	if err != nil {
		return nil, err
	}

	entries := map[string]vault.Entry{}
	switch entryType {
	case "secret":
		entries = data.Secrets
	case "config":
		entries = data.Configs
	default:
		if e, ok := data.Secrets[name]; ok {
			entries[name] = e
		} else if e, ok := data.Configs[name]; ok {
			entries[name] = e
		}
	}

	entry, ok := entries[name]
	if !ok {
		return nil, fmt.Errorf("entry %q not found in source %q", name, src.Name)
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

func secretsOf(d *vault.VaultData) map[string]vault.Entry { return d.Secrets }
func configsOf(d *vault.VaultData) map[string]vault.Entry { return d.Configs }

// resolve implements the fallback chain: env+ctx → env+* → *+*
func resolve(entry vault.Entry, env, context string) (string, bool) {
	if env == "" {
		env = "*"
	}
	if context == "" {
		context = "*"
	}
	if val, ok := entry.Values[vault.MakeLookupKey(env, context)]; ok {
		return val, true
	}
	if context != "*" {
		if val, ok := entry.Values[vault.MakeLookupKey(env, "*")]; ok {
			return val, true
		}
	}
	if env != "*" {
		if val, ok := entry.Values[vault.MakeLookupKey("*", "*")]; ok {
			return val, true
		}
	}
	return "", false
}

func entryToInfo(name, source, entryType string, entry vault.Entry) EntryInfo {
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
		Source:       source,
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

// sourceSortKey places primary first, then alphabetical.
func sourceSortKey(name string) string {
	if name == vault.PrimarySourceName {
		return "\x00" + name
	}
	return "\x01" + name
}
