package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/paulderscheid/git-context/internal/config"
	"github.com/paulderscheid/git-context/internal/git"
	"github.com/paulderscheid/git-context/internal/models"
	"github.com/spf13/cobra"
)

var (
	pruneDryRun bool
	pruneForce  bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old snapshots based on retention policy",
	Long: `Remove snapshots older than the retention period.

The retention policy is configured in ~/.config/context/config.toml:
  [retention]
  days = 90
  preserve_tags = ["important", "security"]

Snapshots with preserve tags will never be pruned.

Example:
  context prune              # Show what would be pruned
  context prune --force      # Actually prune snapshots`,
	RunE: runPrune,
}

func init() {
	rootCmd.AddCommand(pruneCmd)

	pruneCmd.Flags().BoolVar(&pruneDryRun, "dry-run", true, "Show what would be pruned without deleting")
	pruneCmd.Flags().BoolVar(&pruneForce, "force", false, "Actually delete branches (overrides dry-run)")
}

func runPrune(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Get retention settings
	retentionDays := config.GetRetentionDays()
	preserveTags := config.GetPreserveTags()

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	fmt.Printf("Retention policy: %d days\n", retentionDays)
	fmt.Printf("Preserve tags: %v\n", preserveTags)
	fmt.Printf("Cutoff date: %s\n\n", cutoffDate.Format("2006-01-02"))

	// Get all snapshot branches
	branches, err := git.ListBranches("snapshot/*")
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	// Find snapshots to prune
	var toPrune []pruneCandidate
	var toPreserve []pruneCandidate

	for _, branch := range branches {
		info, err := parseSnapshotBranch(branch)
		if err != nil {
			continue
		}

		// Read metadata to check tags
		metaPath := models.MetadataPath(info.Timestamp, info.Topic)
		metaContent, err := gitShow(branch, metaPath)
		var metadata *models.Metadata
		if err == nil {
			var m models.Metadata
			if json.Unmarshal([]byte(metaContent), &m) == nil {
				metadata = &m
			}
		}

		candidate := pruneCandidate{
			Branch:    branch,
			Info:      info,
			Metadata:  metadata,
			Age:       time.Since(info.Timestamp),
			Preserved: false,
		}

		// Check if should be preserved
		if metadata != nil && config.ShouldPreserve(metadata.Tags) {
			candidate.Preserved = true
			candidate.Reason = "has preserve tag"
			toPreserve = append(toPreserve, candidate)
			continue
		}

		// Check if older than retention period
		if info.Timestamp.Before(cutoffDate) {
			reason := fmt.Sprintf("older than %d days", retentionDays)
			candidate.Reason = reason
			toPrune = append(toPrune, candidate)
		} else {
			candidate.Preserved = true
			candidate.Reason = "within retention period"
			toPreserve = append(toPreserve, candidate)
		}
	}

	// Display results
	if len(toPrune) == 0 {
		fmt.Println("No snapshots to prune")
		return nil
	}

	fmt.Printf("Snapshots to prune (%d):\n\n", len(toPrune))
	for _, c := range toPrune {
		fmt.Printf("  %s\n", c.Branch)
		fmt.Printf("    Age:    %s\n", formatDuration(c.Age))
		fmt.Printf("    Reason: %s\n", c.Reason)
		if c.Metadata != nil && len(c.Metadata.Tags) > 0 {
			fmt.Printf("    Tags:   %v\n", c.Metadata.Tags)
		}
		fmt.Println()
	}

	if len(toPreserve) > 0 {
		fmt.Printf("Snapshots to preserve (%d):\n\n", len(toPreserve))
		for _, c := range toPreserve {
			fmt.Printf("  %s\n", c.Branch)
			fmt.Printf("    Age:    %s\n", formatDuration(c.Age))
			fmt.Printf("    Reason: %s\n", c.Reason)
			fmt.Println()
		}
	}

	// Perform deletion if --force is specified
	if pruneForce && !pruneDryRun {
		fmt.Println("Pruning snapshots...")
		for _, c := range toPrune {
			fmt.Printf("  Deleting %s...\n", c.Branch)
			if err := git.DeleteBranch(c.Branch, true); err != nil {
				fmt.Printf("    Error: %v\n", err)
			} else {
				fmt.Printf("    ✓ Deleted\n")
			}
		}
		fmt.Printf("\n✓ Pruned %d snapshot(s)\n", len(toPrune))
	} else {
		fmt.Println("\nThis is a dry run. Use --force to actually prune snapshots.")
	}

	return nil
}

type pruneCandidate struct {
	Branch    string
	Info      snapshotInfo
	Metadata  *models.Metadata
	Age       time.Duration
	Preserved bool
	Reason    string
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days == 0 {
		return "< 1 day"
	}
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
