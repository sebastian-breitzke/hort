package daemon

import (
	"bufio"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/s16e/hort/internal/vault"
)

func primarySetup(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	ref, err := vault.PrimaryRef()
	if err != nil {
		t.Fatalf("PrimaryRef(): %v", err)
	}
	key, err := vault.CreatePrimaryVault([]byte("passphrase"))
	if err != nil {
		t.Fatalf("CreatePrimaryVault(): %v", err)
	}
	if err := vault.SaveSessionFor(ref, key); err != nil {
		t.Fatalf("SaveSessionFor(): %v", err)
	}
}

func startServer(t *testing.T) {
	t.Helper()
	// macOS Unix-socket paths are limited to ~104 bytes. t.TempDir paths are
	// too long, so we mint a shorter socket path under /tmp and rely on the
	// session/vault living in HOME (set by primarySetup).
	f, err := os.CreateTemp("", "hort-daemon-*.sock")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	sockPath := f.Name()
	_ = f.Close()
	_ = os.Remove(sockPath)
	t.Cleanup(func() {
		_ = os.Remove(sockPath)
	})

	// Point client-side lookup at this socket for Dial().
	t.Setenv("HORT_SOCKET_PATH", sockPath)

	srv := NewServer(sockPath)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start(): %v", err)
	}
	go func() {
		_ = srv.Serve()
	}()
	t.Cleanup(func() {
		_ = srv.Close()
	})

	// Wait for socket to accept connections
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.Dial("unix", sockPath)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("daemon never accepted connection")
}

func dialOrFatal(t *testing.T) *Client {
	t.Helper()
	c, err := Dial()
	if err != nil {
		t.Fatalf("Dial(): %v", err)
	}
	return c
}

func TestDaemonSetGetRoundTrip(t *testing.T) {
	primarySetup(t)
	startServer(t)

	c := dialOrFatal(t)
	defer c.Close()

	if _, err := c.Call(Request{Method: MethodSetSecret, Params: map[string]any{
		"name": "foo", "value": "bar", "env": "*", "context": "*",
	}}); err != nil {
		t.Fatalf("set: %v", err)
	}
	resp, err := c.Call(Request{Method: MethodGetSecret, Params: map[string]any{
		"name": "foo", "env": "*", "context": "*",
	}})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v, _ := resp.Result["value"].(string); v != "bar" {
		t.Fatalf("got %q, want bar", v)
	}
}

func TestDaemonConcurrentGet(t *testing.T) {
	primarySetup(t)
	startServer(t)

	seed := dialOrFatal(t)
	if _, err := seed.Call(Request{Method: MethodSetSecret, Params: map[string]any{
		"name": "shared", "value": "payload", "env": "*", "context": "*",
	}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	seed.Close()

	const concurrency = 16
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := Dial()
			if err != nil {
				errCh <- err
				return
			}
			defer c.Close()
			resp, err := c.Call(Request{Method: MethodGetSecret, Params: map[string]any{
				"name": "shared", "env": "*", "context": "*",
			}})
			if err != nil {
				errCh <- err
				return
			}
			if v, _ := resp.Result["value"].(string); v != "payload" {
				errCh <- &mismatchError{got: v}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent get: %v", err)
		}
	}
}

type mismatchError struct{ got string }

func (e *mismatchError) Error() string { return "got " + e.got }

// compile-time use of bufio to avoid unused import when the test evolves
var _ = bufio.NewReader
