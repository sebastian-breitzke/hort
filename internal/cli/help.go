package cli

const HelpText = `Hort — Local secret and config store for humans and AI agents.

USAGE:
  hort <command> [flags]

READ:
  hort --secret <name> [--env <env>]     Get a secret value (stdout only)
  hort --config <name> [--env <env>]     Get a config value (stdout only)

WRITE:
  hort --set-secret <name> --value <v> [--env <env>] [--description <d>]
  hort --set-config <name> --value <v> [--env <env>] [--description <d>]

DISCOVER:
  hort --list [--type secret|config]     List all entries with descriptions
  hort --describe <name>                 Show entry details and environments
  hort status                            Show vault status (locked/unlocked, entry count)

MANAGE:
  hort init [--restore]                  Create new vault or restore with existing passphrase
  hort unlock                            Unlock vault with passphrase
  hort lock                              Lock vault (clear session key)

DELETE:
  hort --delete <name> [--env <env>]     Delete entry or specific environment override

FLAGS:
  --env <name>          Target environment (default: * baseline)
  --value <value>       Value to store
  --description <text>  Human-readable description for discovery
  --type <secret|config> Filter --list by entry type
  --json                Machine-readable JSON output for --list, --describe, status
  --help                Show this help text

AGENT INSTRUCTIONS:
  Hort is designed to be used by AI agents (Claude, Codex, Gemini) as a local
  secret and config store. Here is how to use it:

  1. Discovery: Run ` + "`hort --list`" + ` to see all available secrets and configs.
     Use ` + "`hort --describe <name>`" + ` to see which environments exist for an entry.

  2. Reading: Use ` + "`hort --secret <name>`" + ` or ` + "`hort --config <name>`" + ` to get values.
     Output goes to stdout with no decoration — safe for piping and $() subshells.
     Add ` + "`--env <name>`" + ` for environment-specific values.
     Without --env, the baseline (*) value is returned.

  3. Writing: Use --set-secret or --set-config with --value to store entries.
     Always include --description for discoverability.

  4. Error handling:
     - Exit code 0: success, value on stdout
     - Exit code 1: general error (not found, invalid args), message on stderr
     - Exit code 2: vault is locked — ask user to run ` + "`hort unlock`" + `

  5. Do NOT parse the vault file directly. Always use the CLI.
  6. Do NOT store hort output in files or logs — secrets are ephemeral.
  7. Use --json flag with --list and --describe for structured output.

EXAMPLES:
  # Get a secret for use in a command
  TOKEN=$(hort --secret grafana-token --env prod)
  curl -H "Authorization: Bearer $TOKEN" https://monitoring.example.com

  # Store a new secret with description
  hort --set-secret db-password --value "s3cret" --env prod --description "PostgreSQL password"

  # List all secrets
  hort --list --type secret

  # Check what environments exist for an entry
  hort --describe db-password

  # Use in CI/scripts
  API_URL=$(hort --config api-base-url --env staging)
`
