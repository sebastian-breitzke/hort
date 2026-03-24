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
		flagValue     string
		flagDesc      string
		flagList      bool
		flagDescribe  string
		flagDelete    string
		flagType      string
		flagJSON      bool
	)

	root := &cobra.Command{
		Use:   "hort",
		Short: "Hort — Local secret and config store for humans and AI agents",
		Long:  cli.HelpText,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get secret
			if flagSecret != "" {
				return cli.CmdGetSecret(flagSecret, flagEnv)
			}

			// Get config
			if flagConfig != "" {
				return cli.CmdGetConfig(flagConfig, flagEnv)
			}

			// Set secret
			if flagSetSecret != "" {
				if flagValue == "" {
					return fmt.Errorf("--value is required with --set-secret")
				}
				return cli.CmdSetSecret(flagSetSecret, flagValue, flagEnv, flagDesc)
			}

			// Set config
			if flagSetConfig != "" {
				if flagValue == "" {
					return fmt.Errorf("--value is required with --set-config")
				}
				return cli.CmdSetConfig(flagSetConfig, flagValue, flagEnv, flagDesc)
			}

			// List
			if flagList {
				return cli.CmdList(flagType, flagJSON)
			}

			// Describe
			if flagDescribe != "" {
				return cli.CmdDescribe(flagDescribe, flagJSON)
			}

			// Delete
			if flagDelete != "" {
				return cli.CmdDelete(flagDelete, flagEnv)
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

	// Discovery flags
	root.Flags().BoolVar(&flagList, "list", false, "List all entries")
	root.Flags().StringVar(&flagDescribe, "describe", "", "Show entry details")
	root.Flags().StringVar(&flagType, "type", "", "Filter by type: secret or config")
	root.Flags().BoolVar(&flagJSON, "json", false, "JSON output for list/describe/status")

	// Delete flag
	root.Flags().StringVar(&flagDelete, "delete", "", "Delete an entry or environment override")

	// Subcommands
	root.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create a new vault or restore with existing passphrase",
		RunE: func(cmd *cobra.Command, args []string) error {
			restore, _ := cmd.Flags().GetBool("restore")
			return cli.CmdInit(restore)
		},
	})
	root.Commands()[0].Flags().Bool("restore", false, "Restore with existing passphrase")

	root.AddCommand(&cobra.Command{
		Use:   "unlock",
		Short: "Unlock the vault with your passphrase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.CmdUnlock()
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "lock",
		Short: "Lock the vault (clear session key)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.CmdLock()
		},
	})

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show vault status",
		RunE: func(cmd *cobra.Command, args []string) error {
			j, _ := cmd.Flags().GetBool("json")
			return cli.CmdStatus(j)
		},
	}
	statusCmd.Flags().Bool("json", false, "JSON output")
	root.AddCommand(statusCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
