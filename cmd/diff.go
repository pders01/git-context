package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alpkeskin/gotoon"
	"github.com/pders01/git-context/internal/git"
	"github.com/spf13/cobra"
)

var (
	diffJSON bool
	diffToon bool
)

var diffCmd = &cobra.Command{
	Use:   "diff <timestamp1> <topic1> <timestamp2> <topic2>",
	Short: "Compare two snapshots",
	Long: `Compare two snapshots and show differences in:
  - Metadata (tags, notes, mode)
  - Timestamps
  - Related branches
  - Commits

Example:
  context diff 2025-11-14T2252 initial-reconnaissance 2025-11-14T2252 vulnerability-analysis`,
	Args: cobra.ExactArgs(4),
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)

	diffCmd.Flags().BoolVar(&diffJSON, "json", false, "Output as JSON")
	diffCmd.Flags().BoolVar(&diffToon, "toon", false, "Output in LLM-friendly toon format")
}

type snapshotDiff struct {
	Snapshot1       snapshotSummary `json:"snapshot1"`
	Snapshot2       snapshotSummary `json:"snapshot2"`
	TimeDifference  string          `json:"time_difference"`
	TagsAdded       []string        `json:"tags_added"`
	TagsRemoved     []string        `json:"tags_removed"`
	TagsShared      []string        `json:"tags_shared"`
	ModeChanged     bool            `json:"mode_changed"`
	ModeFrom        string          `json:"mode_from,omitempty"`
	ModeTo          string          `json:"mode_to,omitempty"`
	NotesChanged    bool            `json:"notes_changed"`
	BranchChanged   bool            `json:"branch_changed"`
	CommitChanged   bool            `json:"commit_changed"`
}

type snapshotSummary struct {
	Branch    string    `json:"branch"`
	Topic     string    `json:"topic"`
	Timestamp time.Time `json:"timestamp"`
	Mode      string    `json:"mode"`
	Tags      []string  `json:"tags"`
	Notes     string    `json:"notes"`
	Commit    string    `json:"commit"`
}

func runDiff(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Parse both snapshots
	timestamp1 := args[0]
	topic1 := args[1]
	timestamp2 := args[2]
	topic2 := args[3]

	branch1 := fmt.Sprintf("snapshot/%s/%s", timestamp1, topic1)
	branch2 := fmt.Sprintf("snapshot/%s/%s", timestamp2, topic2)

	if !git.BranchExists(branch1) {
		return fmt.Errorf("snapshot branch does not exist: %s", branch1)
	}
	if !git.BranchExists(branch2) {
		return fmt.Errorf("snapshot branch does not exist: %s", branch2)
	}

	// Load snapshot metadata
	info1, err := parseSnapshotBranch(branch1)
	if err != nil {
		return fmt.Errorf("failed to parse snapshot 1: %w", err)
	}
	info1.LoadMetadata()

	info2, err := parseSnapshotBranch(branch2)
	if err != nil {
		return fmt.Errorf("failed to parse snapshot 2: %w", err)
	}
	info2.LoadMetadata()

	// Build diff
	diff := &snapshotDiff{
		Snapshot1: snapshotSummary{
			Branch:    info1.Branch,
			Topic:     info1.Topic,
			Timestamp: info1.Timestamp,
		},
		Snapshot2: snapshotSummary{
			Branch:    info2.Branch,
			Topic:     info2.Topic,
			Timestamp: info2.Timestamp,
		},
	}

	// Time difference
	timeDiff := info2.Timestamp.Sub(info1.Timestamp)
	if timeDiff < 0 {
		timeDiff = -timeDiff
		diff.TimeDifference = fmt.Sprintf("%s (snapshot2 is older)", formatDuration(timeDiff))
	} else {
		diff.TimeDifference = fmt.Sprintf("%s (snapshot2 is newer)", formatDuration(timeDiff))
	}

	// Compare metadata
	if info1.Metadata != nil {
		diff.Snapshot1.Mode = string(info1.Metadata.Mode)
		diff.Snapshot1.Tags = info1.Metadata.Tags
		diff.Snapshot1.Notes = info1.Metadata.Notes
		diff.Snapshot1.Commit = info1.Metadata.MainCommit
	}

	if info2.Metadata != nil {
		diff.Snapshot2.Mode = string(info2.Metadata.Mode)
		diff.Snapshot2.Tags = info2.Metadata.Tags
		diff.Snapshot2.Notes = info2.Metadata.Notes
		diff.Snapshot2.Commit = info2.Metadata.MainCommit
	}

	// Compare modes
	if diff.Snapshot1.Mode != diff.Snapshot2.Mode {
		diff.ModeChanged = true
		diff.ModeFrom = diff.Snapshot1.Mode
		diff.ModeTo = diff.Snapshot2.Mode
	}

	// Compare notes
	diff.NotesChanged = diff.Snapshot1.Notes != diff.Snapshot2.Notes

	// Compare commits
	diff.CommitChanged = diff.Snapshot1.Commit != diff.Snapshot2.Commit

	// Compare tags
	tagMap1 := make(map[string]bool)
	tagMap2 := make(map[string]bool)

	for _, tag := range diff.Snapshot1.Tags {
		tagMap1[tag] = true
	}
	for _, tag := range diff.Snapshot2.Tags {
		tagMap2[tag] = true
	}

	for tag := range tagMap1 {
		if tagMap2[tag] {
			diff.TagsShared = append(diff.TagsShared, tag)
		} else {
			diff.TagsRemoved = append(diff.TagsRemoved, tag)
		}
	}

	for tag := range tagMap2 {
		if !tagMap1[tag] {
			diff.TagsAdded = append(diff.TagsAdded, tag)
		}
	}

	// Output JSON if requested
	if diffJSON {
		output, err := json.MarshalIndent(diff, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Output Toon if requested
	if diffToon {
		output, err := gotoon.Encode(diff)
		if err != nil {
			return fmt.Errorf("failed to encode Toon: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	// Display human-readable diff
	fmt.Println("Snapshot Comparison")
	fmt.Println("━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Printf("Snapshot 1: %s\n", diff.Snapshot1.Branch)
	fmt.Printf("Snapshot 2: %s\n", diff.Snapshot2.Branch)
	fmt.Println()

	fmt.Printf("Time Difference: %s\n", diff.TimeDifference)
	fmt.Println()

	if diff.ModeChanged {
		fmt.Printf("Mode: %s → %s\n", diff.ModeFrom, diff.ModeTo)
	} else {
		fmt.Printf("Mode: %s (unchanged)\n", diff.Snapshot1.Mode)
	}
	fmt.Println()

	if len(diff.TagsAdded) > 0 || len(diff.TagsRemoved) > 0 {
		fmt.Println("Tags:")
		if len(diff.TagsShared) > 0 {
			fmt.Printf("  Shared:  %v\n", diff.TagsShared)
		}
		if len(diff.TagsAdded) > 0 {
			fmt.Printf("  Added:   %v\n", diff.TagsAdded)
		}
		if len(diff.TagsRemoved) > 0 {
			fmt.Printf("  Removed: %v\n", diff.TagsRemoved)
		}
		fmt.Println()
	} else if len(diff.TagsShared) > 0 {
		fmt.Printf("Tags: %v (unchanged)\n\n", diff.TagsShared)
	}

	if diff.CommitChanged {
		fmt.Printf("Commit: %s → %s\n", diff.Snapshot1.Commit[:8], diff.Snapshot2.Commit[:8])
	} else {
		fmt.Printf("Commit: %s (unchanged)\n", diff.Snapshot1.Commit[:8])
	}
	fmt.Println()

	if diff.NotesChanged {
		fmt.Println("Notes Changed:")
		fmt.Printf("  Snapshot 1: %s\n", truncate(diff.Snapshot1.Notes, 100))
		fmt.Printf("  Snapshot 2: %s\n", truncate(diff.Snapshot2.Notes, 100))
	} else {
		fmt.Println("Notes: (unchanged)")
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
