package vault

import (
	"os"
	"testing"
	"time"
)

func TestWriteLockBlocksConcurrentWriter(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	unlock, err := lockVault()
	if err != nil {
		t.Fatalf("lockVault(): %v", err)
	}
	defer func() {
		_ = unlock()
	}()

	acquired := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		nextUnlock, err := lockVault()
		if err != nil {
			errCh <- err
			return
		}
		defer func() {
			_ = nextUnlock()
		}()
		close(acquired)
	}()

	select {
	case err := <-errCh:
		t.Fatalf("concurrent lock failed: %v", err)
	case <-acquired:
		t.Fatal("concurrent writer acquired lock before release")
	case <-time.After(150 * time.Millisecond):
	}

	if err := unlock(); err != nil {
		t.Fatalf("unlock(): %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("concurrent lock failed after release: %v", err)
	case <-acquired:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent writer did not acquire lock after release")
	}
}

func TestSaveVaultCreatesBackup(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	key, err := CreateVault([]byte("passphrase"))
	if err != nil {
		t.Fatalf("CreateVault(): %v", err)
	}

	data, raw, err := LoadVault(key)
	if err != nil {
		t.Fatalf("LoadVault(): %v", err)
	}

	data.Secrets["demo"] = Entry{
		Description: "demo",
		Values: map[LookupKey]string{
			MakeLookupKey("dev", "spuerhund"): "value",
		},
	}

	if err := SaveVault(data, key, raw); err != nil {
		t.Fatalf("SaveVault(): %v", err)
	}

	backupPath, err := BackupPath()
	if err != nil {
		t.Fatalf("BackupPath(): %v", err)
	}

	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("stat backup: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("backup file is empty")
	}
}
