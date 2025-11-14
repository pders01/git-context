package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/pders01/git-context/internal/git"
	"github.com/pders01/git-context/internal/models"
	"github.com/spf13/cobra"
)

var (
	openPath string
)

var openCmd = &cobra.Command{
	Use:   "open <timestamp> <topic>",
	Short: "Open a snapshot in a worktree",
	Long: `Create a git worktree and open the snapshot for viewing.

This creates a separate working directory for the snapshot without
affecting your current workspace.

Example:
  context open 2025-11-14T0930 security-audit

This creates a worktree at ../snap-0930 by default.`,
	Args: cobra.ExactArgs(2),
	RunE: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)

	openCmd.Flags().StringVar(&openPath, "path", "", "Custom worktree path (default: ../snap-HHMM)")
}

func runOpen(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	timestampStr := args[0]
	topic := args[1]

	// Parse timestamp
	timestamp, err := time.Parse("2006-01-02T1504", timestampStr)
	if err != nil {
		return fmt.Errorf("invalid timestamp format (use YYYY-MM-DDTHHMM): %w", err)
	}

	// Build branch name
	branch := models.BranchName(timestamp, topic)

	// Check if branch exists
	if !git.BranchExists(branch) {
		return fmt.Errorf("snapshot branch does not exist: %s", branch)
	}

	// Determine worktree path
	worktreePath := openPath
	if worktreePath == "" {
		timeStr := timestamp.Format("1504")
		worktreePath = filepath.Join("..", fmt.Sprintf("snap-%s", timeStr))
	}

	fmt.Printf("Creating worktree for: %s\n", branch)
	fmt.Printf("Path: %s\n", worktreePath)

	// Create worktree
	if err := git.CreateWorktree(worktreePath, branch); err != nil {
		return err
	}

	fmt.Printf("\n✓ Worktree created at: %s\n", worktreePath)
	fmt.Printf("  cd %s\n", worktreePath)

	return nil
}
