package main

import (
	"fmt"
	"os"

	"github.com/s16e/hort/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	var (
		flagSecret    string
		flagConfig    string
		flagSetSecret string
		flagSetConfig string
		flagEnv       string
		flagContext   string
		flagValue     string
		flagDesc      string
		flagList      bool
		flagDescribe  string
		flagDelete    string
		flagType      string
		flagJSON      bool
		flagSource    string
	)

	root := &cobra.Command{
		Use:   "hort",
		Short: "Hort — Local secret and config store for humans and AI agents",
		Long:  cli.HelpText,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagSecret != "" {
				return cli.CmdGetSecret(flagSecret, flagEnv, flagContext, flagSource)
			}
			if flagConfig != "" {
				return cli.CmdGetConfig(flagConfig, flagEnv, flagContext, flagSource)
			}
			if flagSetSecret != "" {
				if flagValue == "" {
					return fmt.Errorf("--value is required with --set-secret")
				}
				return cli.CmdSetSecret(flagSetSecret, flagValue, flagEnv, flagContext, flagDesc, flagSource)
			}
			if flagSetConfig != "" {
				if flagValue == "" {
					return fmt.Errorf("--value is required with --set-config")
				}
				return cli.CmdSetConfig(flagSetConfig, flagValue, flagEnv, flagContext, flagDesc, flagSource)
			}
			if flagList {
				return cli.CmdList(flagType, flagJSON)
			}
			if flagDescribe != "" {
				return cli.CmdDescribe(flagDescribe, flagJSON)
			}
			if flagDelete != "" {
				return cli.CmdDelete(flagDelete, flagEnv, flagContext, flagSource)
			}
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Read flags
	root.Flags().StringVar(&flagSecret, "secret", "", "Get a secret value")
	root.Flags().StringVar(&flagConfig, "config", "", "Get a config value")

	// Write flags
	root.Flags().StringVar(&flagSetSecret, "set-secret", "", "Store a secret")
	root.Flags().StringVar(&flagSetConfig, "set-config", "", "Store a config")
	root.Flags().StringVar(&flagValue, "value", "", "Value to store")
	root.Flags().StringVar(&flagDesc, "description", "", "Description for discovery")

	// Common flags
	root.Flags().StringVar(&flagEnv, "env", "", "Target environment (default: * baseline)")
	root.Flags().StringVar(&flagContext, "context", "", "Target context, e.g. tenant/customer (default: * baseline)")
	root.Flags().StringVar(&flagSource, "source", "", "Target a specific source (primary by default for writes; required when reads are ambiguous)")

	// Discovery flags
	root.Flags().BoolVar(&flagList, "list", false, "List all entries")
	root.Flags().StringVar(&flagDescribe, "describe", "", "Show entry details")
	root.Flags().StringVar(&flagType, "type", "", "Filter by type: secret or config")
	root.Flags().BoolVar(&flagJSON, "json", false, "JSON output for list/describe/status")

	// Delete flag
	root.Flags().StringVar(&flagDelete, "delete", "", "Delete an entry or specific env/context override")

	// Subcommands
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new primary vault or restore with existing passphrase",
		RunE: func(cmd *cobra.Command, args []string) error {
			restore, _ := cmd.Flags().GetBool("restore")
			return cli.CmdInit(restore)
		},
	}
	initCmd.Flags().Bool("restore", false, "Restore with existing passphrase")
	root.AddCommand(initCmd)

	root.AddCommand(&cobra.Command{
		Use:   "unlock",
		Short: "Unlock the primary vault with your passphrase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.CmdUnlock()
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "lock",
		Short: "Lock the primary vault (clear its session key)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.CmdLock()
		},
	})

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show primary vault status",
		RunE: func(cmd *cobra.Command, args []string) error {
			j, _ := cmd.Flags().GetBool("json")
			return cli.CmdStatus(j)
		},
	}
	statusCmd.Flags().Bool("json", false, "JSON output")
	root.AddCommand(statusCmd)

	// Source subcommands: mount / unmount / list
	sourceCmd := &cobra.Command{
		Use:   "source",
		Short: "Manage mounted secret sources",
	}

	mountCmd := &cobra.Command{
		Use:   "mount",
		Short: "Register a mounted source vault and cache its key",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			path, _ := cmd.Flags().GetString("path")
			keyHex, _ := cmd.Flags().GetString("key-hex")
			kdf, _ := cmd.Flags().GetString("kdf")
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if keyHex == "" {
				return fmt.Errorf("--key-hex is required")
			}
			return cli.CmdSourceMount(name, path, keyHex, kdf)
		},
	}
	mountCmd.Flags().String("name", "", "Source name (unique)")
	mountCmd.Flags().String("path", "", "Vault file path")
	mountCmd.Flags().String("key-hex", "", "32-byte key, hex-encoded (64 hex chars)")
	mountCmd.Flags().String("kdf", "raw", "KDF mode: 'raw' (default) or 'argon2id'")
	sourceCmd.AddCommand(mountCmd)

	unmountCmd := &cobra.Command{
		Use:   "unmount",
		Short: "Remove a mounted source (vault file stays on disk)",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			return cli.CmdSourceUnmount(name)
		},
	}
	unmountCmd.Flags().String("name", "", "Source name")
	sourceCmd.AddCommand(unmountCmd)

	sourceListCmd := &cobra.Command{
		Use:   "list",
		Short: "List known sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			j, _ := cmd.Flags().GetBool("json")
			return cli.CmdSourceList(j)
		},
	}
	sourceListCmd.Flags().Bool("json", false, "JSON output")
	sourceCmd.AddCommand(sourceListCmd)

	root.AddCommand(sourceCmd)

	// Daemon subcommands
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the Hort background daemon (Unix socket)",
	}
	daemonCmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Run the daemon in the foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.CmdDaemonStart()
		},
	})
	daemonCmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Send SIGTERM to a running daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.CmdDaemonStop()
		},
	})
	daemonStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check whether the daemon socket is responsive",
		RunE: func(cmd *cobra.Command, args []string) error {
			j, _ := cmd.Flags().GetBool("json")
			return cli.CmdDaemonStatus(j)
		},
	}
	daemonStatusCmd.Flags().Bool("json", false, "JSON output")
	daemonCmd.AddCommand(daemonStatusCmd)
	root.AddCommand(daemonCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
