# Hort

Local secret and config store for humans and AI agents.

![Hort Logo](logo-placeholder.png)

## What

Hort is a cross-platform CLI that stores secrets and configs in a single encrypted vault file. Designed for AI agents (Claude, Codex, Gemini) to discover and read credentials without touching files directly.

One tool. One vault. All environments.

## Install

### macOS (Homebrew)

```bash
brew install s16e/tap/hort
```

### Windows (Scoop)

```bash
scoop bucket add s16e https://github.com/s16e/scoop-bucket
scoop install hort
```

### Binary Download

Grab the latest release from [GitHub Releases](https://github.com/s16e/hort/releases).

### From Source

```bash
go install github.com/s16e/hort/cmd/hort@latest
```

## Quick Start

```bash
# Create your vault
hort init

# Store a secret
hort --set-secret db-password --value "s3cret" --description "PostgreSQL password"

# Store an environment-specific override
hort --set-secret db-password --value "prod-s3cret" --env prod

# Read it back
hort --secret db-password           # → s3cret (baseline)
hort --secret db-password --env prod  # → prod-s3cret

# Store configs (non-sensitive but environment-aware)
hort --set-config api-url --value "https://api.example.com" --description "Base API URL"
hort --set-config api-url --value "https://api.prod.example.com" --env prod

# Discover what's available
hort --list
hort --describe db-password

# Use in scripts
TOKEN=$(hort --secret grafana-token --env prod)
curl -H "Authorization: Bearer $TOKEN" https://monitoring.example.com
```

## How It Works

### Vault

Everything lives in `~/.hort/vault.enc` — a single AES-256-GCM encrypted file. The encryption key is derived from your passphrase via Argon2id (memory-hard, GPU-resistant).

### Environments

Every entry supports environment-specific values with `*` as the baseline:

```bash
hort --set-secret api-key --value "test-key"              # baseline (*)
hort --set-secret api-key --value "prod-key" --env prod   # production override
hort --secret api-key --env prod                           # → prod-key
hort --secret api-key --env staging                        # → test-key (falls back to *)
```

### Session

Run `hort unlock` once — the derived key is cached in `~/.hort/.session` (chmod 600). Stays unlocked until you run `hort lock`. No daemon, no background process.

## CLI Reference

### Read

```bash
hort --secret <name> [--env <env>]    # Raw value on stdout, no decoration
hort --config <name> [--env <env>]    # Same for configs
```

### Write

```bash
hort --set-secret <name> --value <val> [--env <env>] [--description <desc>]
hort --set-config <name> --value <val> [--env <env>] [--description <desc>]
```

### Discover

```bash
hort --list [--type secret|config] [--json]
hort --describe <name> [--json]
hort status [--json]
```

### Manage

```bash
hort init [--restore]     # Create vault or restore with existing passphrase
hort unlock               # Unlock vault
hort lock                 # Lock vault
```

### Delete

```bash
hort --delete <name> [--env <env>]    # Delete entry or specific environment
```

## Agent Integration

Hort is built for AI agents. The `--help` output includes agent-specific instructions. Agents should:

1. Run `hort --list` to discover available secrets/configs
2. Run `hort --describe <name>` to see environments
3. Read values with `hort --secret <name>` or `hort --config <name>`
4. Output is raw on stdout — safe for `$()` and pipes
5. Exit code 2 means vault is locked — ask the user to run `hort unlock`

### Automation

Set `HORT_PASSPHRASE` to skip interactive passphrase prompts (useful for CI/CD):

```bash
export HORT_PASSPHRASE="my-passphrase"
hort init    # no prompt
hort unlock  # no prompt
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (not found, invalid args) |
| 2 | Vault locked |

## Security

- AES-256-GCM authenticated encryption
- Argon2id key derivation (memory-hard, GPU-resistant)
- Session key: file permissions (chmod 600), OS user-scoped
- No network access, no telemetry, no cloud sync
- Threat model: protects against other OS users. Root access = game over (accepted for a local tool)

## Name

**Hort** (German) — the Nibelungenhort, a legendary treasure hoard. Where the treasure lies.

## License

MIT
