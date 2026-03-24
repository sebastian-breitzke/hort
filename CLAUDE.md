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
- `internal/vault/crypto.go` — AES-256-GCM, Argon2id
- `internal/vault/vault.go` — Vault file operations
- `internal/vault/session.go` — Session key management
- `internal/store/store.go` — High-level get/set/list/delete
- `internal/cli/commands.go` — Command implementations
- `internal/cli/help.go` — Help text (serves as agent prompt)
- `internal/cli/output.go` — Output formatting (plain/JSON)

## Conventions

- Errors on stderr, values on stdout (no decoration)
- Exit code 2 = vault locked
- `HORT_PASSPHRASE` env var bypasses terminal prompt
