package store

import (
	"fmt"
	"sync"
	"testing"

	"github.com/s16e/hort/internal/vault"
)

func TestConcurrentMutationsKeepAllEntries(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	key, err := vault.CreateVault([]byte("passphrase"))
	if err != nil {
		t.Fatalf("CreateVault(): %v", err)
	}
	if err := vault.SaveSession(key); err != nil {
		t.Fatalf("SaveSession(): %v", err)
	}

	const writers = 32

	var wg sync.WaitGroup
	errCh := make(chan error, writers)

	for i := 0; i < writers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := NewFromSession()
			if err != nil {
				errCh <- err
				return
			}

			name := fmt.Sprintf("entry-%02d", i)
			value := fmt.Sprintf("value-%02d", i)
			if i%2 == 0 {
				err = s.SetSecret(name, value, "dev", "spuerhund", "test entry")
			} else {
				err = s.SetConfig(name, value, "dev", "spuerhund", "test entry")
			}
			if err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent mutation failed: %v", err)
		}
	}

	s, err := NewFromSession()
	if err != nil {
		t.Fatalf("NewFromSession(): %v", err)
	}

	for i := 0; i < writers; i++ {
		name := fmt.Sprintf("entry-%02d", i)
		expected := fmt.Sprintf("value-%02d", i)
		if i%2 == 0 {
			value, err := s.GetSecret(name, "dev", "spuerhund")
			if err != nil {
				t.Fatalf("GetSecret(%s): %v", name, err)
			}
			if value != expected {
				t.Fatalf("GetSecret(%s) = %q, want %q", name, value, expected)
			}
		} else {
			value, err := s.GetConfig(name, "dev", "spuerhund")
			if err != nil {
				t.Fatalf("GetConfig(%s): %v", name, err)
			}
			if value != expected {
				t.Fatalf("GetConfig(%s) = %q, want %q", name, value, expected)
			}
		}
	}
}
