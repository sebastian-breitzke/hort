# Hort

Local secret and config store CLI. Go project.

## Build

```bash
go build ./cmd/hort/
```

## Test

```bash
go test ./...
```

## Run

```bash
# Init vault (needs terminal or HORT_PASSPHRASE env var)
HORT_PASSPHRASE="test" ./hort init

# Store and retrieve
./hort --set-secret foo --value "bar" --description "test secret"
./hort --secret foo

# Full help
./hort --help
```

## Structure

- `cmd/hort/main.go` — Entry point, Cobra CLI wiring
- `internal/vault/crypto.go` — AES-256-GCM, Argon2id KDF, v1/v2 format helpers
- `internal/vault/format.go` — v2 header layout + KDF flag byte (raw vs Argon2id)
- `internal/vault/vault.go` — `VaultRef`, load/save/update by ref, primary helpers
- `internal/vault/sources.go` — Mounted-source registry (~/.hort/sources/index.json)
- `internal/vault/session.go` — Per-ref session key persistence
- `internal/vault/lock.go` — Per-ref flock (concurrent-safe writers)
- `internal/store/store.go` — Multi-source merge reads, targeted writes, ambiguity detection
- `internal/daemon/*.go` — Unix-socket daemon (server, client, protocol, paths)
- `internal/cli/commands.go` — Command implementations (transparently prefer daemon)
- `internal/cli/daemon.go` — `hort daemon start/stop/status`
- `internal/cli/service.go` — `hort daemon install/uninstall` (launchd + systemd template rendering)
- `internal/cli/help.go` — Help text (serves as agent prompt)
- `internal/cli/output.go` — Output formatting (plain/JSON)
- `packaging/packaging.go` — `//go:embed` wrappers exposing the service templates
- `packaging/launchd/tech.s16e.hort.daemon.plist.tmpl` — rendered by `hort daemon install` on macOS
- `packaging/systemd/hort-daemon.service.tmpl` — rendered by `hort daemon install` on Linux

## Sources

Primary vault at `~/.hort/vault.enc` (Argon2id KDF). Any number of mounted
sources registered via `hort source mount --name <n> --path <p> --key-hex <hex>`;
their vault files live wherever the mounter decides, their lock/session files
under `~/.hort/sources/<name>.{lock,session}`. Reads merge across all unlocked
sources. Writes default to primary; `--source <name>` targets a mount. Same
key in multiple sources → ambiguity error with `--source` hint.

## Daemon

`hort daemon start` launches a foreground Unix-socket server at
`~/.hort/daemon.sock` (override via `HORT_SOCKET_PATH` for tests — macOS
limits socket paths to ~104 bytes). When the socket is up, every CLI call
detects it and routes the request through the daemon instead of reopening
vault files. Fallback to direct file access is automatic when the daemon is
stopped.

`hort daemon install` renders the embedded launchd plist (macOS) or systemd
user unit (Linux) with `os.Executable()` and `$HOME`, drops it at
`~/Library/LaunchAgents/` or `~/.config/systemd/user/`, and activates it with
`launchctl bootstrap` / `systemctl --user enable --now`. Idempotent. `--system`
switches to `/Library/LaunchDaemons` / `/etc/systemd/system` (root only).
`hort daemon uninstall` reverses both. Windows prints "not supported".

## Release

Tag push triggers GoReleaser (binaries + Homebrew) automatically. npm publish is manual:

```bash
git tag v<version> && git push --tags
cd npm && npm version <version> --no-git-tag-version && npm publish --access=public
```

Package: `@s16e/hort`. Do not automate npm publish via CI.

## Conventions

- Errors on stderr, values on stdout (no decoration)
- Exit code 2 = primary vault locked (mounts that are locked are skipped with a stderr warning)
- `HORT_PASSPHRASE` env var bypasses terminal prompt
- `HORT_SOCKET_PATH` env var overrides the daemon socket path (tests only)
- `primary` is a reserved source name; mount names must match `[a-z0-9._-]+` and not start with `-` or `.`
