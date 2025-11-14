package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/paulderscheid/git-context/internal/git"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize context in the current repository",
	Long: `Install git hooks and create configuration for context snapshots.

This command:
  - Installs the pre-commit hook to prevent modifications to snapshots
  - Creates a default config file if it doesn't exist

Run this once per repository to set up context snapshot protection.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Get the hooks directory
	hooksDir := ".git/hooks"
	if _, err := os.Stat(hooksDir); os.IsNotExist(err) {
		return fmt.Errorf("git hooks directory not found: %s", hooksDir)
	}

	// Install pre-commit hook
	hookContent := `#!/bin/sh
# Git hook to prevent commits on snapshot branches
# This ensures snapshot immutability

branch=$(git rev-parse --abbrev-ref HEAD)

if echo "$branch" | grep -q "^snapshot/"; then
    echo "ERROR: Snapshots are immutable. Cannot commit to snapshot branch: $branch"
    echo "Create a new snapshot instead with: context save <topic>"
    exit 1
fi

exit 0
`

	hookPath := filepath.Join(hooksDir, "pre-commit")

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil {
		fmt.Printf("Warning: pre-commit hook already exists at %s\n", hookPath)
		fmt.Println("To preserve immutability, ensure it includes snapshot branch protection.")

		// TODO: Could offer to append our check to existing hook
		return nil
	}

	// Write the hook
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return fmt.Errorf("failed to write pre-commit hook: %w", err)
	}

	fmt.Printf("✓ Installed pre-commit hook: %s\n", hookPath)
	fmt.Println("  Snapshot branches are now protected from modifications")

	// Create default config if it doesn't exist
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "context")
	configPath := filepath.Join(configDir, "config.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create config directory
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Create default config
		defaultConfig := `[retention]
days = 90
preserve_tags = ["important", "security", "architecture"]

[snapshot]
default_mode = "full"
research_dir = "research"
`

		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}

		fmt.Printf("✓ Created default config: %s\n", configPath)
	} else {
		fmt.Printf("Config already exists: %s\n", configPath)
	}

	fmt.Println("\n✓ Context initialized successfully!")
	fmt.Println("  You can now use: context save <topic>")

	return nil
}
