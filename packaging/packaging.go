// Package packaging exposes embedded service-manager templates so the hort
// CLI can render and install them at runtime. The templates themselves live
// under packaging/launchd and packaging/systemd so humans can read them
// directly and so macOS cask / Linux distro packagers can ship the same
// canonical copies.
package packaging

import _ "embed"

//go:embed launchd/tech.s16e.hort.daemon.plist.tmpl
var LaunchdPlistTemplate string

//go:embed systemd/hort-daemon.service.tmpl
var SystemdServiceTemplate string

// LaunchdLabel is the canonical launchd label for the Hort daemon. Used as
// both the file stem (Label.plist) and the service identifier passed to
// launchctl.
const LaunchdLabel = "tech.s16e.hort.daemon"

// SystemdUnitName is the canonical systemd unit name.
const SystemdUnitName = "hort-daemon.service"
