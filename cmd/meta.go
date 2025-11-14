package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/pders01/git-context/internal/git"
	"github.com/pders01/git-context/internal/models"
	"github.com/spf13/cobra"
)

var metaCmd = &cobra.Command{
	Use:   "meta <timestamp> <topic>",
	Short: "Show metadata for a snapshot",
	Long: `Display the metadata (meta.json) for a specific snapshot.

Example:
  context meta 2025-11-14T0930 security-audit`,
	Args: cobra.ExactArgs(2),
	RunE: runMeta,
}

func init() {
	rootCmd.AddCommand(metaCmd)
}

func runMeta(cmd *cobra.Command, args []string) error {
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

	// Get metadata path
	metaPath := models.MetadataPath(timestamp, topic)

	// Read metadata using git show
	metaContent, err := gitShow(branch, metaPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Parse metadata
	var metadata models.Metadata
	if err := json.Unmarshal([]byte(metaContent), &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Display metadata
	fmt.Printf("Snapshot: %s\n\n", branch)
	fmt.Printf("Topic:         %s\n", metadata.Topic)
	fmt.Printf("Created:       %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Mode:          %s\n", metadata.Mode)
	fmt.Printf("Related Branch: %s\n", metadata.RelatedBranch)
	fmt.Printf("Main Commit:   %s\n", metadata.MainCommit)

	if len(metadata.Tags) > 0 {
		fmt.Printf("Tags:          %v\n", metadata.Tags)
	}

	if metadata.TreeHash != "" {
		fmt.Printf("Tree Hash:     %s\n", metadata.TreeHash)
	}

	if metadata.Embedding != "" {
		fmt.Printf("Embedding:     %s\n", metadata.Embedding)
	}

	if metadata.Notes != "" {
		fmt.Printf("\nNotes:\n%s\n", metadata.Notes)
	}

	return nil
}

// gitShow reads a file from a specific branch using git show
func gitShow(branch, path string) (string, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", branch, path))
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
