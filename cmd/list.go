package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/paulderscheid/git-context/internal/git"
	"github.com/paulderscheid/git-context/internal/models"
	"github.com/spf13/cobra"
)

var (
	listTopic string
	listToday bool
	listSince string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all context snapshots",
	Long: `List all context snapshots with optional filtering.

Examples:
  context list
  context list --topic security
  context list --today
  context list --since 2025-10-01`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listTopic, "topic", "", "Filter by topic")
	listCmd.Flags().BoolVar(&listToday, "today", false, "Show only today's snapshots")
	listCmd.Flags().StringVar(&listSince, "since", "", "Show snapshots since date (YYYY-MM-DD)")
}

func runList(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Get all snapshot branches
	branches, err := git.ListBranches("snapshot/*")
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	// Parse and filter snapshots
	var snapshots []snapshotInfo
	for _, branch := range branches {
		info, err := parseSnapshotBranch(branch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse branch %s: %v\n", branch, err)
			continue
		}

		// Apply filters
		if listTopic != "" && info.Topic != listTopic {
			continue
		}

		if listToday {
			today := time.Now().Format("2006-01-02")
			snapshotDate := info.Timestamp.Format("2006-01-02")
			if today != snapshotDate {
				continue
			}
		}

		if listSince != "" {
			sinceDate, err := time.Parse("2006-01-02", listSince)
			if err != nil {
				return fmt.Errorf("invalid --since date format (use YYYY-MM-DD): %w", err)
			}
			if info.Timestamp.Before(sinceDate) {
				continue
			}
		}

		snapshots = append(snapshots, info)
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots match the filter criteria")
		return nil
	}

	// Sort by timestamp (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	// Display snapshots
	fmt.Printf("Found %d snapshot(s):\n\n", len(snapshots))
	for _, s := range snapshots {
		fmt.Printf("  %s\n", s.Branch)
		fmt.Printf("    Topic:   %s\n", s.Topic)
		fmt.Printf("    Created: %s\n", s.Timestamp.Format("2006-01-02 15:04"))

		// Try to load metadata for additional info
		if meta := s.LoadMetadata(); meta != nil {
			fmt.Printf("    Mode:    %s\n", meta.Mode)
			if len(meta.Tags) > 0 {
				fmt.Printf("    Tags:    %v\n", meta.Tags)
			}
			if meta.Notes != "" {
				notes := meta.Notes
				if len(notes) > 60 {
					notes = notes[:60] + "..."
				}
				fmt.Printf("    Notes:   %s\n", notes)
			}
		}
		fmt.Println()
	}

	return nil
}

type snapshotInfo struct {
	Branch    string
	Timestamp time.Time
	Topic     string
}

func (s *snapshotInfo) LoadMetadata() *models.Metadata {
	metaPath := models.MetadataPath(s.Timestamp, s.Topic)

	// We need to check out the branch to read the file
	// For now, we'll skip this to avoid branch switching
	// TODO: Use git show to read file without checkout
	_ = metaPath
	return nil
}

func parseSnapshotBranch(branch string) (snapshotInfo, error) {
	// Format: snapshot/YYYY-MM-DDTHHMM/topic-slug
	parts := strings.Split(branch, "/")
	if len(parts) != 3 {
		return snapshotInfo{}, fmt.Errorf("invalid snapshot branch format")
	}

	if parts[0] != "snapshot" {
		return snapshotInfo{}, fmt.Errorf("not a snapshot branch")
	}

	timestamp, err := time.Parse("2006-01-02T1504", parts[1])
	if err != nil {
		return snapshotInfo{}, fmt.Errorf("invalid timestamp format: %w", err)
	}

	return snapshotInfo{
		Branch:    branch,
		Timestamp: timestamp,
		Topic:     parts[2],
	}, nil
}

// readMetadataFromBranch reads metadata from a branch without checking it out
func readMetadataFromBranch(branch string, metaPath string) (*models.Metadata, error) {
	// Use git show to read file content
	// This is a helper function for future use
	_ = branch
	_ = metaPath

	// For now, return nil
	// TODO: Implement using git show
	return nil, nil
}
