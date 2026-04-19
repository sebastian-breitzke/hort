package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/s16e/hort/internal/daemon"
)

// CmdDaemonStart runs the daemon in the foreground. Users who want background
// operation should wrap this with their own process manager (launchd/systemd),
// or run it under `nohup ... &`.
func CmdDaemonStart() error {
	sockPath, err := daemon.SocketPath()
	if err != nil {
		return err
	}

	// Refuse to start if another daemon already owns the socket.
	if daemon.Available() {
		return fmt.Errorf("daemon already running (socket %s responsive)", sockPath)
	}

	pidPath, err := daemon.PIDPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		return fmt.Errorf("writing pid file: %w", err)
	}
	defer os.Remove(pidPath)

	srv := daemon.NewServer(sockPath)
	if err := srv.Start(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Hort daemon listening on %s (pid %d)\n", sockPath, os.Getpid())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, shutdownSignals()...)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()

	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "Received %s, shutting down.\n", sig)
	case err := <-errCh:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}

	return srv.Close()
}

// CmdDaemonStop reads the pid file and sends SIGTERM.
func CmdDaemonStop() error {
	pidPath, err := daemon.PIDPath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("no pid file (is the daemon running?): %w", err)
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil {
		return fmt.Errorf("invalid pid file: %w", err)
	}
	if err := signalDaemonStop(pid); err != nil {
		return fmt.Errorf("signaling daemon: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Sent stop signal to daemon pid %d.\n", pid)
	return nil
}

// CmdDaemonStatus reports whether the daemon socket is responsive.
func CmdDaemonStatus(jsonOutput bool) error {
	sockPath, err := daemon.SocketPath()
	if err != nil {
		return err
	}
	running := daemon.Available()

	if jsonOutput {
		payload := map[string]any{
			"running":     running,
			"socket_path": sockPath,
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	state := "stopped"
	if running {
		state = "running"
	}
	fmt.Printf("Daemon:  %s\nSocket:  %s\n", state, sockPath)
	return nil
}
