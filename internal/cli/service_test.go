package cli

import (
	"strings"
	"testing"
)

func TestRenderLaunchdPlistUser(t *testing.T) {
	got, err := RenderLaunchdPlist(ServiceParams{
		BinaryPath: "/opt/homebrew/bin/hort",
		Label:      "tech.s16e.hort.daemon",
		Home:       "/Users/alice",
		HortDir:    "/Users/alice/.hort",
		StdoutLog:  "/Users/alice/.hort/daemon.out.log",
		StderrLog:  "/Users/alice/.hort/daemon.err.log",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, want := range []string{
		"<string>tech.s16e.hort.daemon</string>",
		"<string>/opt/homebrew/bin/hort</string>",
		"<string>daemon</string>",
		"<string>start</string>",
		"<key>HOME</key>",
		"<string>/Users/alice</string>",
		"<key>WorkingDirectory</key>",
		"<string>/Users/alice/.hort/daemon.out.log</string>",
		"<string>/Users/alice/.hort/daemon.err.log</string>",
		"<key>RunAtLoad</key>",
		"<key>KeepAlive</key>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in rendered plist:\n%s", want, got)
		}
	}

	// The template must not leave raw placeholders behind.
	if strings.Contains(got, "{{") || strings.Contains(got, "}}") {
		t.Errorf("unrendered template markers present:\n%s", got)
	}
}

func TestRenderLaunchdPlistSystem(t *testing.T) {
	got, err := RenderLaunchdPlist(ServiceParams{
		BinaryPath: "/usr/local/bin/hort",
		Label:      "tech.s16e.hort.daemon",
		HortDir:    "/var/log",
		StdoutLog:  "/var/log/hort-daemon.out.log",
		StderrLog:  "/var/log/hort-daemon.err.log",
		System:     true,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// No HOME / WorkingDirectory when installed system-wide (no single user).
	for _, unwanted := range []string{
		"<key>HOME</key>",
		"<key>WorkingDirectory</key>",
	} {
		if strings.Contains(got, unwanted) {
			t.Errorf("system install should not include %q:\n%s", unwanted, got)
		}
	}

	if !strings.Contains(got, "<string>/usr/local/bin/hort</string>") {
		t.Errorf("binary path missing:\n%s", got)
	}
}

func TestRenderSystemdUnitUser(t *testing.T) {
	got, err := RenderSystemdUnit(ServiceParams{
		BinaryPath: "/home/alice/.local/bin/hort",
		Home:       "/home/alice",
		WantedBy:   "default.target",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, want := range []string{
		"ExecStart=/home/alice/.local/bin/hort daemon start",
		"Environment=HOME=/home/alice",
		"WantedBy=default.target",
		"Restart=on-failure",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in rendered unit:\n%s", want, got)
		}
	}
}

func TestRenderSystemdUnitSystem(t *testing.T) {
	got, err := RenderSystemdUnit(ServiceParams{
		BinaryPath: "/usr/bin/hort",
		WantedBy:   "multi-user.target",
		System:     true,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(got, "Environment=HOME=") {
		t.Errorf("system install should omit HOME:\n%s", got)
	}
	if !strings.Contains(got, "WantedBy=multi-user.target") {
		t.Errorf("missing WantedBy=multi-user.target:\n%s", got)
	}
}
