package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/s16e/hort/internal/store"
)

// FormatList formats entry list for human or JSON output.
func FormatList(entries []store.EntryInfo, jsonOutput bool) string {
	if jsonOutput {
		data, _ := json.MarshalIndent(entries, "", "  ")
		return string(data)
	}

	if len(entries) == 0 {
		return "No entries found.\n"
	}

	var b strings.Builder
	for _, e := range entries {
		envs := strings.Join(e.Environments, ", ")
		desc := e.Description
		if desc == "" {
			desc = "(no description)"
		}

		line := fmt.Sprintf("%-8s %-30s %s  [env: %s]", e.Type, e.Name, desc, envs)
		if len(e.Contexts) > 0 {
			line += fmt.Sprintf("  [ctx: %s]", strings.Join(e.Contexts, ", "))
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// FormatDescribe formats entry details for human or JSON output.
func FormatDescribe(entry *store.EntryInfo, jsonOutput bool) string {
	if jsonOutput {
		data, _ := json.MarshalIndent(entry, "", "  ")
		return string(data)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Name:         %s\n", entry.Name)
	fmt.Fprintf(&b, "Type:         %s\n", entry.Type)
	fmt.Fprintf(&b, "Description:  %s\n", entry.Description)
	fmt.Fprintf(&b, "Environments: %s\n", strings.Join(entry.Environments, ", "))
	if len(entry.Contexts) > 0 {
		fmt.Fprintf(&b, "Contexts:     %s\n", strings.Join(entry.Contexts, ", "))
	}
	return b.String()
}

// FormatStatus formats vault status for human or JSON output.
func FormatStatus(unlocked bool, vaultPath string, secretCount, configCount int, jsonOutput bool) string {
	if jsonOutput {
		status := map[string]any{
			"unlocked":     unlocked,
			"vault_path":   vaultPath,
			"secret_count": secretCount,
			"config_count": configCount,
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		return string(data)
	}

	var b strings.Builder
	if unlocked {
		fmt.Fprintln(&b, "Status:   unlocked")
	} else {
		fmt.Fprintln(&b, "Status:   locked")
	}
	fmt.Fprintf(&b, "Vault:    %s\n", vaultPath)
	fmt.Fprintf(&b, "Secrets:  %d\n", secretCount)
	fmt.Fprintf(&b, "Configs:  %d\n", configCount)
	return b.String()
}

// FormatContextValues formats context→value map for human or JSON output.
func FormatContextValues(values map[string]string, jsonOutput bool) string {
	if jsonOutput {
		data, _ := json.MarshalIndent(values, "", "  ")
		return string(data)
	}

	var b strings.Builder
	for ctx, val := range values {
		fmt.Fprintf(&b, "%-20s %s\n", ctx, val)
	}
	return b.String()
}
