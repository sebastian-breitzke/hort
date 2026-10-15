package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"text/template"

	"github.com/s16e/hort/internal/daemon"
	"github.com/s16e/hort/internal/vault"
	"github.com/s16e/hort/packaging"
)

// ServiceParams is the data passed to the launchd / systemd templates.
// Exported so template-rendering tests can drive it directly.
type ServiceParams struct {
	BinaryPath string
	Label      string
	Home       string
	HortDir    string
	StdoutLog  string
	StderrLog  string
	WantedBy   string
	System     bool
}

// RenderLaunchdPlist returns the fully rendered launchd plist.
func RenderLaunchdPlist(p ServiceParams) (string, error) {
	return renderTemplate("launchd", packaging.LaunchdPlistTemplate, p)
}

// RenderSystemdUnit returns the fully rendered systemd unit file.
func RenderSystemdUnit(p ServiceParams) (string, error) {
	return renderTemplate("systemd", packaging.SystemdServiceTemplate, p)
}

func renderTemplate(name, body string, data any) (string, error) {
	t, err := template.New(name).Parse(body)
	if err != nil {
		return "", fmt.Errorf("parsing %s template: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering %s template: %w", name, err)
	}
	return buf.String(), nil
}

// resolveBinary returns the absolute, symlink-resolved path of the running hort
// binary. `launchctl` and `systemd` both want a concrete path in the unit file.
func resolveBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating hort binary: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}

func paramsForUserInstall() (ServiceParams, error) {
	bin, err := resolveBinary()
	if err != nil {
		return ServiceParams{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ServiceParams{}, fmt.Errorf("finding home directory: %w", err)
	}
	hortDir, err := vault.HortDir()
	if err != nil {
		return ServiceParams{}, err
	}
	return ServiceParams{
		BinaryPath: bin,
		Label:      packaging.LaunchdLabel,
		Home:       home,
		HortDir:    hortDir,
		StdoutLog:  filepath.Join(hortDir, "daemon.out.log"),
		StderrLog:  filepath.Join(hortDir, "daemon.err.log"),
		WantedBy:   "default.target",
		System:     false,
	}, nil
}

func paramsForSystemInstall() (ServiceParams, error) {
	bin, err := resolveBinary()
	if err != nil {
		return ServiceParams{}, err
	}
	return ServiceParams{
		BinaryPath: bin,
		Label:      packaging.LaunchdLabel,
		Home:       "",
		HortDir:    "/var/log",
		StdoutLog:  "/var/log/hort-daemon.out.log",
		StderrLog:  "/var/log/hort-daemon.err.log",
		WantedBy:   "multi-user.target",
		System:     true,
	}, nil
}

// CmdDaemonInstall writes the service definition for the current platform and
// activates it. Idempotent: if the service is already loaded, it is reloaded
// with the freshly rendered file.
func CmdDaemonInstall(systemWide bool) error {
	switch runtime.GOOS {
	case "darwin":
		return installLaunchd(systemWide)
	case "linux":
		return installSystemd(systemWide)
	default:
		return fmt.Errorf("`hort daemon install` is not supported on %s yet — run `hort daemon start` manually or wrap it in your own supervisor", runtime.GOOS)
	}
}

// CmdDaemonUninstall removes and unloads the service definition. Does not
// error if nothing is installed.
func CmdDaemonUninstall(systemWide bool) error {
	switch runtime.GOOS {
	case "darwin":
		return uninstallLaunchd(systemWide)
	case "linux":
		return uninstallSystemd(systemWide)
	default:
		return fmt.Errorf("`hort daemon uninstall` is not supported on %s", runtime.GOOS)
	}
}

// ---- launchd (macOS) -------------------------------------------------------

func launchdPlistPath(system bool) string {
	name := packaging.LaunchdLabel + ".plist"
	if system {
		return filepath.Join("/Library/LaunchDaemons", name)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", name)
}

func launchdDomain(system bool) (string, error) {
	if system {
		return "system", nil
	}
	return "gui/" + strconv.Itoa(os.Getuid()), nil
}

func installLaunchd(system bool) error {
	var params ServiceParams
	var err error
	if system {
		params, err = paramsForSystemInstall()
	} else {
		params, err = paramsForUserInstall()
	}
	if err != nil {
		return err
	}

	rendered, err := RenderLaunchdPlist(params)
	if err != nil {
		return err
	}

	plistPath := launchdPlistPath(system)
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(plistPath), err)
	}
	if err := writeFileAtomic(plistPath, []byte(rendered), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}
	if !system {
		// Make sure the log directory exists so launchd can open the files.
		if err := os.MkdirAll(params.HortDir, 0700); err != nil {
			return fmt.Errorf("creating %s: %w", params.HortDir, err)
		}
	}

	domain, err := launchdDomain(system)
	if err != nil {
		return err
	}
	// Bootout any previous version (ignore "not loaded" errors), then bootstrap.
	_ = runQuiet("launchctl", "bootout", domain+"/"+packaging.LaunchdLabel)
	if err := run("launchctl", "bootstrap", domain, plistPath); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w", err)
	}
	_ = runQuiet("launchctl", "enable", domain+"/"+packaging.LaunchdLabel)

	fmt.Fprintf(os.Stderr, "Installed launchd agent %s\n", plistPath)
	fmt.Fprintf(os.Stderr, "Binary:   %s\n", params.BinaryPath)
	fmt.Fprintf(os.Stderr, "Socket:   %s\n", socketPathOrBlank())
	return nil
}

func uninstallLaunchd(system bool) error {
	plistPath := launchdPlistPath(system)
	domain, err := launchdDomain(system)
	if err != nil {
		return err
	}
	_ = runQuiet("launchctl", "bootout", domain+"/"+packaging.LaunchdLabel)

	if _, err := os.Stat(plistPath); err == nil {
		if err := os.Remove(plistPath); err != nil {
			return fmt.Errorf("removing %s: %w", plistPath, err)
		}
		fmt.Fprintf(os.Stderr, "Removed %s\n", plistPath)
	} else if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "No launchd agent installed.")
	} else {
		return err
	}
	return nil
}

// ---- systemd (Linux) -------------------------------------------------------

func systemdUnitPath(system bool) (string, error) {
	if system {
		return filepath.Join("/etc/systemd/system", packaging.SystemdUnitName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user", packaging.SystemdUnitName), nil
}

func systemctlArgs(system bool, rest ...string) []string {
	if system {
		return rest
	}
	return append([]string{"--user"}, rest...)
}

func installSystemd(system bool) error {
	var params ServiceParams
	var err error
	if system {
		params, err = paramsForSystemInstall()
	} else {
		params, err = paramsForUserInstall()
	}
	if err != nil {
		return err
	}

	rendered, err := RenderSystemdUnit(params)
	if err != nil {
		return err
	}

	unitPath, err := systemdUnitPath(system)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(unitPath), err)
	}
	if err := writeFileAtomic(unitPath, []byte(rendered), 0644); err != nil {
		return fmt.Errorf("writing unit: %w", err)
	}
	if !system {
		if err := os.MkdirAll(params.HortDir, 0700); err != nil {
			return fmt.Errorf("creating %s: %w", params.HortDir, err)
		}
	}

	if err := run("systemctl", systemctlArgs(system, "daemon-reload")...); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	if err := run("systemctl", systemctlArgs(system, "enable", "--now", packaging.SystemdUnitName)...); err != nil {
		return fmt.Errorf("systemctl enable --now: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Installed systemd unit %s\n", unitPath)
	fmt.Fprintf(os.Stderr, "Binary:   %s\n", params.BinaryPath)
	fmt.Fprintf(os.Stderr, "Socket:   %s\n", socketPathOrBlank())
	return nil
}

func uninstallSystemd(system bool) error {
	unitPath, err := systemdUnitPath(system)
	if err != nil {
		return err
	}

	_ = runQuiet("systemctl", systemctlArgs(system, "disable", "--now", packaging.SystemdUnitName)...)

	if _, err := os.Stat(unitPath); err == nil {
		if err := os.Remove(unitPath); err != nil {
			return fmt.Errorf("removing %s: %w", unitPath, err)
		}
		fmt.Fprintf(os.Stderr, "Removed %s\n", unitPath)
	} else if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "No systemd unit installed.")
	} else {
		return err
	}

	_ = runQuiet("systemctl", systemctlArgs(system, "daemon-reload")...)
	return nil
}

// ---- small helpers ---------------------------------------------------------

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".hort-install-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

func socketPathOrBlank() string {
	if p, err := daemon.SocketPath(); err == nil {
		return p
	}
	return ""
}

