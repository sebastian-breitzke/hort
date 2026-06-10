//go:build !windows

package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/s16e/hort/internal/vault"
)

// TestDaemonSlowSourceDoesNotStallOtherRequests reproduces the live incident
// from 2026-06-10: a mounted source vault on a sleeping external volume hangs
// in open(2), and with a daemon-global dispatch mutex every other request
// stalled behind it. A FIFO stands in for the slow volume — os.ReadFile blocks
// on open until a writer appears, exactly like a disk spin-up.
func TestDaemonSlowSourceDoesNotStallOtherRequests(t *testing.T) {
	primarySetup(t)

	home, _ := os.UserHomeDir()
	fifoPath := filepath.Join(home, "slow.hort.enc")
	if err := unix.Mkfifo(fifoPath, 0600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	slowRef, err := vault.MountedRefAt("slow", fifoPath)
	if err != nil {
		t.Fatalf("MountedRefAt(): %v", err)
	}
	if err := vault.SaveSessionFor(slowRef, make([]byte, 32)); err != nil {
		t.Fatalf("SaveSessionFor(): %v", err)
	}
	if err := vault.AddSource(vault.SourceRecord{Name: "slow", Path: fifoPath, KDF: vault.KDFRawKey}); err != nil {
		t.Fatalf("AddSource(): %v", err)
	}

	startServer(t)

	seed := dialOrFatal(t)
	if _, err := seed.Call(Request{Method: MethodSetSecret, Params: map[string]any{
		"name": "shared", "value": "payload", "env": "*", "context": "*",
	}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	seed.Close()

	// A merged read touches every source and blocks opening the FIFO.
	mergedDone := make(chan error, 1)
	go func() {
		c, err := Dial()
		if err != nil {
			mergedDone <- err
			return
		}
		defer c.Close()
		_, err = c.Call(Request{Method: MethodGetSecret, Params: map[string]any{
			"name": "shared", "env": "*", "context": "*",
		}})
		mergedDone <- err
	}()

	// Opening the FIFO write side blocks until the daemon's ReadFile has opened
	// the read side — once this returns, the merged request is provably stuck
	// inside vault.LoadVault.
	writer, err := os.OpenFile(fifoPath, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open fifo writer: %v", err)
	}

	// While the merged read hangs, a primary-targeted read must still complete.
	primaryDone := make(chan error, 1)
	go func() {
		c, err := Dial()
		if err != nil {
			primaryDone <- err
			return
		}
		defer c.Close()
		resp, err := c.Call(Request{Method: MethodGetSecret, Params: map[string]any{
			"name": "shared", "env": "*", "context": "*", "source": "primary",
		}})
		if err == nil {
			if v, _ := resp.Result["value"].(string); v != "payload" {
				err = &mismatchError{got: v}
			}
		}
		primaryDone <- err
	}()

	select {
	case err := <-primaryDone:
		if err != nil {
			t.Fatalf("primary read during slow source open: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("primary read stalled behind slow source open")
	}

	// Release the merged read: EOF on the FIFO makes the slow source unreadable
	// (skipped with a warning), so the merged read resolves from primary.
	_ = writer.Close()
	select {
	case err := <-mergedDone:
		if err != nil {
			t.Fatalf("merged read after slow source released: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("merged read never completed after releasing the slow source")
	}
}
