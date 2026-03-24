package cli

const HelpText = `Hort — Local secret and config store for humans and AI agents.

USAGE:
  hort <command> [flags]

READ:
  hort --secret <name> [--env <env>] [--context <ctx>]   Get a secret value (stdout only)
  hort --config <name> [--env <env>] [--context <ctx>]   Get a config value (stdout only)

WRITE:
  hort --set-secret <name> --value <v> [--env <env>] [--context <ctx>] [--description <d>]
  hort --set-config <name> --value <v> [--env <env>] [--context <ctx>] [--description <d>]

DISCOVER:
  hort --list [--type secret|config]     List all entries with descriptions, environments, and contexts
  hort --describe <name>                 Show entry details: environments, contexts, description
  hort status                            Show vault status (locked/unlocked, entry count)

MANAGE:
  hort init [--restore]                  Create new vault or restore with existing passphrase
  hort unlock                            Unlock vault with passphrase
  hort lock                              Lock vault (clear session key)

DELETE:
  hort --delete <name> [--env <env>] [--context <ctx>]   Delete entry or specific override

FLAGS:
  --env <name>          Target environment (default: * baseline)
  --context <name>      Target context, e.g. tenant or customer (default: * baseline)
  --value <value>       Value to store
  --description <text>  Human-readable description for discovery
  --type <secret|config> Filter --list by entry type
  --json                Machine-readable JSON output for --list, --describe, status
  --help                Show this help text

ENVIRONMENTS AND CONTEXTS:
  Hort supports two dimensions for value lookup:
  - Environment (--env): dev, int, prod, etc.
  - Context (--context): tenant, customer, project — any second dimension you need.

  Values are stored with a combined key of env:context.
  Without --env or --context, the baseline (*) is used for that dimension.

  Fallback chain when reading:
    env+context → env+* → *+*

  Example:
    hort --set-secret tenant-id --value "default-123"
    hort --set-secret tenant-id --value "prod-123" --env prod
    hort --set-secret tenant-id --value "heine-prod-456" --env prod --context heine
    hort --set-secret tenant-id --value "otto-prod-789" --env prod --context otto

    hort --secret tenant-id --env prod --context heine   → heine-prod-456
    hort --secret tenant-id --env prod --context otto    → otto-prod-789
    hort --secret tenant-id --env prod                   → prod-123 (no context = *)
    hort --secret tenant-id --env staging                → default-123 (fallback to *+*)

AGENT INSTRUCTIONS:
  Hort is designed to be used by AI agents (Claude, Codex, Gemini) as a local
  secret and config store. Here is how to use it:

  1. Discovery: Run ` + "`hort --list`" + ` to see all available secrets and configs.
     Use ` + "`hort --describe <name>`" + ` to see environments and contexts for an entry.

  2. Reading: Use ` + "`hort --secret <name>`" + ` or ` + "`hort --config <name>`" + ` to get values.
     Output goes to stdout with no decoration — safe for piping and $() subshells.
     Add ` + "`--env <name>`" + ` for environment-specific values.
     Add ` + "`--context <name>`" + ` for context-specific values (e.g. tenant, customer).
     Without --env/--context, baseline (*) values are returned.

  3. Writing: Use --set-secret or --set-config with --value to store entries.
     Always include --description for discoverability.

  4. Contexts: If ` + "`hort --describe <name>`" + ` shows contexts (e.g. heine, otto),
     you may need to ask the user which context to use, or pick the right one
     based on the current task.

  5. Error handling:
     - Exit code 0: success, value on stdout
     - Exit code 1: general error (not found, invalid args), message on stderr
     - Exit code 2: vault is locked — ask user to run ` + "`hort unlock`" + `

  6. Do NOT parse the vault file directly. Always use the CLI.
  7. Do NOT store hort output in files or logs — secrets are ephemeral.
  8. Use --json flag with --list and --describe for structured output.

EXAMPLES:
  # Get a secret for use in a command
  TOKEN=$(hort --secret grafana-token --env prod)
  curl -H "Authorization: Bearer $TOKEN" https://monitoring.example.com

  # Store a context-specific secret
  hort --set-secret tenant-id --value "abc-123" --env prod --context heine --description "SimpleChain tenant ID"

  # Discover what's available
  hort --list --type secret
  hort --describe tenant-id

  # Use in CI/scripts
  API_URL=$(hort --config api-base-url --env staging)
`
