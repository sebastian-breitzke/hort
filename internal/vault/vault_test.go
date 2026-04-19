package vault

import (
	"os"
	"testing"
	"time"
)

func primaryRefOrFatal(t *testing.T) VaultRef {
	t.Helper()
	ref, err := PrimaryRef()
	if err != nil {
		t.Fatalf("PrimaryRef(): %v", err)
	}
	return ref
}

func TestWriteLockBlocksConcurrentWriter(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ref := primaryRefOrFatal(t)

	unlock, err := lockVault(ref)
	if err != nil {
		t.Fatalf("lockVault(): %v", err)
	}
	defer func() {
		_ = unlock()
	}()

	acquired := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		nextUnlock, err := lockVault(ref)
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
	ref := primaryRefOrFatal(t)

	key, err := CreatePrimaryVault([]byte("passphrase"))
	if err != nil {
		t.Fatalf("CreatePrimaryVault(): %v", err)
	}

	data, raw, err := LoadVault(ref, key)
	if err != nil {
		t.Fatalf("LoadVault(): %v", err)
	}

	data.Secrets["demo"] = Entry{
		Description: "demo",
		Values: map[LookupKey]string{
			MakeLookupKey("dev", "spuerhund"): "value",
		},
	}

	if err := SaveVault(ref, data, key, raw); err != nil {
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

func TestCreateV2RawKeyRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ref, err := MountedRefAt("test", t.TempDir()+"/test.enc")
	if err != nil {
		t.Fatalf("MountedRefAt(): %v", err)
	}

	rawKey := make([]byte, KeySize)
	for i := range rawKey {
		rawKey[i] = byte(i)
	}

	sessionKey, err := CreateVault(ref, rawKey, KDFRawKey)
	if err != nil {
		t.Fatalf("CreateVault(raw): %v", err)
	}

	// Raw-key session == material
	for i := range rawKey {
		if sessionKey[i] != rawKey[i] {
			t.Fatalf("raw session key mismatch at %d: %x vs %x", i, sessionKey[i], rawKey[i])
		}
	}

	data, raw, err := LoadVault(ref, sessionKey)
	if err != nil {
		t.Fatalf("LoadVault(): %v", err)
	}
	data.Secrets["k"] = Entry{Values: map[LookupKey]string{MakeLookupKey("*", "*"): "v"}}
	if err := SaveVault(ref, data, sessionKey, raw); err != nil {
		t.Fatalf("SaveVault(): %v", err)
	}

	// Unlock via material path
	key2, err := UnlockVault(ref, rawKey)
	if err != nil {
		t.Fatalf("UnlockVault(): %v", err)
	}
	data2, _, err := LoadVault(ref, key2)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if data2.Secrets["k"].Values[MakeLookupKey("*", "*")] != "v" {
		t.Fatalf("expected round-trip value, got %+v", data2.Secrets)
	}
}

func TestCreatePrimaryVaultArgonRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ref := primaryRefOrFatal(t)

	pass := []byte("correct horse battery staple")
	key, err := CreatePrimaryVault(pass)
	if err != nil {
		t.Fatalf("CreatePrimaryVault(): %v", err)
	}

	// Fresh unlock recovers the key
	key2, err := UnlockVault(ref, pass)
	if err != nil {
		t.Fatalf("UnlockVault(): %v", err)
	}

	data, _, err := LoadVault(ref, key2)
	if err != nil {
		t.Fatalf("LoadVault(): %v", err)
	}
	_ = data // empty but decryptable

	_ = key
}
