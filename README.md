# Hort

**Local secret and config store for humans and AI agents.**

![Hort Logo](logo-placeholder.png)

## Why

I work with three AI coding agents daily — Claude Code, Codex, Gemini. All of them need credentials. API tokens, database passwords, tenant IDs, service URLs. Across environments. Across customers.

Here is how that works today: `.env` files scattered across projects. Environment variables that vanish between sessions. Hardcoded values in configs that shouldn't be committed. Or the agent asks you, you paste it in, it forgets, it asks again next session.

None of this is discoverable. An agent can't ask "what secrets are available?" — it has to know upfront or ask you every time. None of it is environment-aware — switching between dev and prod means manual juggling. None of it is secure — plaintext files with credentials sitting in project directories. And none of it works across agents — each tool has its own way of handling credentials, or doesn't handle them at all.

*Hort* — the Nibelungenhort, where the treasure lies — solves this. One encrypted vault, one CLI, all agents, all environments.

## Install

Copy-paste the script for your OS. It downloads the latest release, installs the binary, and walks you through creating your vault.

### macOS / Linux

```bash
curl -fsSL https://github.com/sebastian-breitzke/hort/releases/latest/download/hort_$(uname -s)_$(uname -m).tar.gz \
  | tar -xz -C /tmp hort \
  && sudo mv /tmp/hort /usr/local/bin/hort \
  && echo "✓ hort installed at $(which hort)" \
  && hort init
```

### macOS (Homebrew)

```bash
brew install sebastian-breitzke/tap/hort && hort init
```

### Windows (PowerShell)

```powershell
$url = "https://github.com/sebastian-breitzke/hort/releases/latest/download/hort_Windows_x86_64.zip"
$tmp = "$env:TEMP\hort.zip"
Invoke-WebRequest -Uri $url -OutFile $tmp
Expand-Archive -Path $tmp -DestinationPath "$env:LOCALAPPDATA\hort" -Force
$env:PATH += ";$env:LOCALAPPDATA\hort"
[Environment]::SetEnvironmentVariable("PATH", "$([Environment]::GetEnvironmentVariable('PATH', 'User'));$env:LOCALAPPDATA\hort", "User")
Remove-Item $tmp
Write-Host "✓ hort installed" -ForegroundColor Green
hort init
```

### Windows (Scoop)

```powershell
scoop bucket add sebastian-breitzke https://github.com/sebastian-breitzke/scoop-bucket
scoop install hort
hort init
```

### From Source

```bash
go install github.com/sebastian-breitzke/hort/cmd/hort@latest && hort init
```

After `hort init` you'll set a passphrase — that's it. Vault created, session unlocked, ready to store secrets.

## Usage

### Getting Started

```bash
# Create your vault — you'll set a passphrase
hort init

# Store your first secret
hort --set-secret grafana-token --value "glsa_..." --description "Grafana API token"

# Use it
TOKEN=$(hort --secret grafana-token)
curl -H "Authorization: Bearer $TOKEN" https://monitoring.example.com
```

### Environments

Every entry supports environment-specific values. Without `--env`, the baseline (`*`) is used.

```bash
hort --set-secret db-password --value "dev-pass" --description "PostgreSQL password"
hort --set-secret db-password --value "prod-pass" --env prod

hort --secret db-password              # → dev-pass (baseline)
hort --secret db-password --env prod   # → prod-pass
hort --secret db-password --env staging # → dev-pass (fallback to baseline)
```

### Contexts

Secrets aren't just per-environment. They can be per-environment *and* per-context. A context is any second dimension: tenant, customer, project.

```bash
hort --set-secret tenant-id --value "default-123" --description "SimpleChain tenant ID"
hort --set-secret tenant-id --value "prod-123" --env prod
hort --set-secret tenant-id --value "heine-456" --env prod --context heine
hort --set-secret tenant-id --value "otto-789" --env prod --context otto

hort --secret tenant-id --env prod --context heine  # → heine-456
hort --secret tenant-id --env prod --context otto   # → otto-789
hort --secret tenant-id --env prod                  # → prod-123
hort --secret tenant-id --env staging               # → default-123 (fallback to *+*)
```

Fallback chain: `env+context → env+* → *+*`. Specific wins, baseline catches the rest.

### Configs

Non-sensitive values that still need environment/context awareness:

```bash
hort --set-config api-url --value "https://api.dev.example.com" --description "Base API URL"
hort --set-config api-url --value "https://api.example.com" --env prod

API_URL=$(hort --config api-url --env prod)
```

### Discovery

Agents need to know what's available. Hort makes everything discoverable:

```bash
hort --list
# config   api-url      Base API URL           [env: *, prod]
# secret   tenant-id    SimpleChain tenant ID  [env: *, prod]  [ctx: heine, otto]

hort --describe tenant-id
# Name:         tenant-id
# Type:         secret
# Description:  SimpleChain tenant ID
# Environments: *, prod
# Contexts:     heine, otto

hort --list --json   # Machine-readable output
hort --describe tenant-id --json
```

### Vault Management

```bash
hort init              # Create new vault (set passphrase)
hort init --restore    # Restore with existing passphrase on a new machine
hort unlock            # Unlock vault (passphrase → session key cached)
hort lock              # Lock vault (clear session key)
hort status            # Show vault status, entry counts
```

### Delete

```bash
hort --delete tenant-id                          # Delete entire entry
hort --delete tenant-id --env prod               # Delete prod override only
hort --delete tenant-id --env prod --context heine  # Delete specific env+context
```

## Agent Integration

Hort is agent-first, not agent-compatible. The `--help` output includes an `AGENT INSTRUCTIONS` section — when an agent runs `hort --help`, it learns the full interface in one call.

**How agents use Hort:**

1. `hort --list` — discover available secrets and configs
2. `hort --describe <name>` — see environments and contexts for an entry
3. `hort --secret <name>` / `hort --config <name>` — read values
4. Output is raw on stdout, no decoration — safe for `$()` and pipes
5. `--json` flag for structured output when agents prefer to parse
6. Exit code 2 = vault locked → agent asks user to run `hort unlock`

**Automation:** Set `HORT_PASSPHRASE` to skip interactive prompts (CI/CD, scripting):

```bash
export HORT_PASSPHRASE="my-passphrase"
hort init    # no terminal prompt
hort unlock  # no terminal prompt
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (not found, invalid args) |
| 2 | Vault is locked |

## Implementation Details

### Architecture

```
cmd/hort/main.go            CLI entry point (Cobra)
internal/
├── vault/
│   ├── crypto.go            AES-256-GCM encrypt/decrypt, Argon2id key derivation
│   ├── vault.go             Vault file operations, data model
│   └── session.go           Session key persistence
├── store/
│   └── store.go             High-level get/set/list/delete with fallback resolution
└── cli/
    ├── commands.go          Command implementations
    ├── help.go              Help text (doubles as agent prompt)
    └── output.go            Output formatting (plain text + JSON)
```

### Encryption

Everything lives in a single file: `~/.hort/vault.enc`.

**File format** (binary):

```
[salt:32 bytes][argon2id-time:4][argon2id-memory:4][argon2id-threads:1][nonce:12][ciphertext+tag:...]
```

- **Key derivation:** Argon2id (memory-hard, GPU-resistant) with 64 MB memory cost, 3 iterations, 4 threads
- **Encryption:** AES-256-GCM with authenticated encryption (tamper-proof)
- **Salt:** 32 bytes, cryptographically random, generated at vault creation
- **Nonce:** 12 bytes, fresh random nonce on every write

The header (salt + Argon2id params) is unencrypted by design — these are not secrets. The ciphertext is the AES-256-GCM sealed JSON vault body.

### Vault Structure (decrypted)

```json
{
  "version": 1,
  "secrets": {
    "tenant-id": {
      "description": "SimpleChain tenant ID",
      "values": {
        "*:*": "default-123",
        "prod:*": "prod-123",
        "prod:heine": "heine-456",
        "prod:otto": "otto-789"
      }
    }
  },
  "configs": {
    "api-url": {
      "description": "Base API URL",
      "values": {
        "*:*": "https://api.dev.example.com",
        "prod:*": "https://api.example.com"
      }
    }
  }
}
```

Values are keyed by `env:context`. `*` is the baseline for either dimension.

### Session

`hort unlock` derives the encryption key from your passphrase and caches it in `~/.hort/.session` (chmod 600, OS user-owned). The session persists until `hort lock` — no TTL, no daemon.

Subsequent CLI calls read the session key and use it to decrypt the vault on every operation. No in-memory state between calls.

### Security Model

- **Encryption:** AES-256-GCM — authenticated, tamper-proof
- **Key derivation:** Argon2id — memory-hard, resistant to GPU/ASIC attacks
- **Session key:** File-based, chmod 600, scoped to OS user
- **No network:** Zero network access, no telemetry, no cloud sync, no phone-home
- **Threat model:** Protects against other OS users and casual file access. Root access = game over — accepted for a local development tool
- **Portability:** Same passphrase decrypts the vault on any machine. Copy `vault.enc`, run `hort init --restore`

### Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `golang.org/x/crypto` | Argon2id key derivation |
| `golang.org/x/term` | Terminal passphrase input (no echo) |
| Standard library | `crypto/aes`, `crypto/cipher`, `crypto/rand`, `encoding/json`, `os` |

### Cross-Platform

Single Go binary. No CGO, no system dependencies. Cross-compiles to:

- macOS (amd64, arm64)
- Linux (amd64, arm64)
- Windows (amd64)

GoReleaser builds all targets automatically on GitHub Release.

## Name

**Hort** (German, /hɔʁt/) — the Nibelungenhort, the legendary treasure hoard from Germanic mythology. Where the gold lies. Where your secrets belong.

## License

MIT
