package cli

import (
	"fmt"
	"os"

	"github.com/s16e/hort/internal/store"
	"github.com/s16e/hort/internal/vault"
	"golang.org/x/term"
)

// ExitLocked is the exit code when the vault is locked.
const ExitLocked = 2

// ReadPassphrase reads a passphrase from HORT_PASSPHRASE env var or terminal (no echo).
func ReadPassphrase(prompt string) ([]byte, error) {
	if env := os.Getenv("HORT_PASSPHRASE"); env != "" {
		return []byte(env), nil
	}

	fmt.Fprint(os.Stderr, prompt)
	pass, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("reading passphrase: %w", err)
	}
	return pass, nil
}

// CmdInit creates a new vault or restores from an existing passphrase.
func CmdInit(restore bool) error {
	exists, err := vault.VaultExists()
	if err != nil {
		return err
	}

	if exists && !restore {
		return fmt.Errorf("vault already exists — use `hort init --restore` to restore with existing passphrase")
	}

	pass, err := ReadPassphrase("Enter passphrase: ")
	if err != nil {
		return err
	}

	if !restore {
		confirm, err := ReadPassphrase("Confirm passphrase: ")
		if err != nil {
			return err
		}
		if string(pass) != string(confirm) {
			return fmt.Errorf("passphrases do not match")
		}
	}

	if restore && exists {
		key, err := vault.UnlockVault(pass)
		if err != nil {
			return err
		}
		if err := vault.SaveSession(key); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Vault restored and unlocked.")
		return nil
	}

	key, err := vault.CreateVault(pass)
	if err != nil {
		return err
	}
	if err := vault.SaveSession(key); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Vault created and unlocked.")
	return nil
}

// CmdUnlock unlocks the vault.
func CmdUnlock() error {
	exists, err := vault.VaultExists()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("no vault found — run `hort init` first")
	}

	if vault.IsUnlocked() {
		fmt.Fprintln(os.Stderr, "Vault is already unlocked.")
		return nil
	}

	pass, err := ReadPassphrase("Enter passphrase: ")
	if err != nil {
		return err
	}

	key, err := vault.UnlockVault(pass)
	if err != nil {
		return err
	}

	if err := vault.SaveSession(key); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Vault unlocked.")
	return nil
}

// CmdLock locks the vault.
func CmdLock() error {
	if err := vault.ClearSession(); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Vault locked.")
	return nil
}

// CmdStatus shows vault status.
func CmdStatus(jsonOutput bool) error {
	unlocked := vault.IsUnlocked()
	path, _ := vault.VaultPath()

	secretCount := 0
	configCount := 0

	if unlocked {
		s, err := store.NewFromSession()
		if err == nil {
			entries, err := s.List("")
			if err == nil {
				for _, e := range entries {
					if e.Type == "secret" {
						secretCount++
					} else {
						configCount++
					}
				}
			}
		}
	}

	fmt.Print(FormatStatus(unlocked, path, secretCount, configCount, jsonOutput))
	return nil
}

// CmdGetSecret retrieves a secret.
func CmdGetSecret(name, env, context string) error {
	s, err := store.NewFromSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitLocked)
	}

	val, err := s.GetSecret(name, env, context)
	if err != nil {
		return err
	}

	fmt.Print(val)
	return nil
}

// CmdGetConfig retrieves a config.
func CmdGetConfig(name, env, context string) error {
	s, err := store.NewFromSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitLocked)
	}

	val, err := s.GetConfig(name, env, context)
	if err != nil {
		return err
	}

	fmt.Print(val)
	return nil
}

// CmdSetSecret stores a secret.
func CmdSetSecret(name, value, env, context, description string) error {
	s, err := store.NewFromSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitLocked)
	}

	if err := s.SetSecret(name, value, env, context, description); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Secret stored.")
	return nil
}

// CmdSetConfig stores a config.
func CmdSetConfig(name, value, env, context, description string) error {
	s, err := store.NewFromSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitLocked)
	}

	if err := s.SetConfig(name, value, env, context, description); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Config stored.")
	return nil
}

// CmdList lists entries.
func CmdList(typeFilter string, jsonOutput bool) error {
	s, err := store.NewFromSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitLocked)
	}

	entries, err := s.List(typeFilter)
	if err != nil {
		return err
	}

	fmt.Print(FormatList(entries, jsonOutput))
	return nil
}

// CmdDescribe shows entry details.
func CmdDescribe(name string, jsonOutput bool) error {
	s, err := store.NewFromSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitLocked)
	}

	entry, err := s.Describe(name)
	if err != nil {
		return err
	}

	fmt.Print(FormatDescribe(entry, jsonOutput))
	return nil
}

// CmdDelete removes an entry or specific env/context combination.
func CmdDelete(name, env, context string) error {
	s, err := store.NewFromSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitLocked)
	}

	if err := s.Delete(name, env, context); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Deleted.")
	return nil
}
