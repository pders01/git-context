package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/alpkeskin/gotoon"
	"github.com/pders01/git-context/internal/git"
	"github.com/spf13/cobra"
)

var (
	relatedJSON bool
	relatedToon bool
)

var relatedCmd = &cobra.Command{
	Use:   "related <timestamp> <topic>",
	Short: "Find related snapshots",
	Long: `Find snapshots related to a given snapshot based on:
  - Explicit relationships (related_to field)
  - Shared tags
  - Same topic

Results are ranked by relevance.

Example:
  context related 2025-11-14T2252 phase-3-complete`,
	Args: cobra.ExactArgs(2),
	RunE: runRelated,
}

func init() {
	rootCmd.AddCommand(relatedCmd)

	relatedCmd.Flags().BoolVar(&relatedJSON, "json", false, "Output as JSON")
	relatedCmd.Flags().BoolVar(&relatedToon, "toon", false, "Output in LLM-friendly toon format")
}

type relatedSnapshot struct {
	Snapshot   snapshotInfo `json:"snapshot"`
	Score      int          `json:"score"`
	Reason     string       `json:"reason"`
}

func runRelated(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	timestampStr := args[0]
	topic := args[1]

	// Parse the target snapshot
	targetBranch := fmt.Sprintf("snapshot/%s/%s", timestampStr, topic)
	if !git.BranchExists(targetBranch) {
		return fmt.Errorf("snapshot branch does not exist: %s", targetBranch)
	}

	// Load target snapshot metadata
	targetInfo, err := parseSnapshotBranch(targetBranch)
	if err != nil {
		return fmt.Errorf("failed to parse snapshot: %w", err)
	}
	targetInfo.LoadMetadata()

	if targetInfo.Metadata == nil {
		return fmt.Errorf("snapshot has no metadata")
	}

	// Get all snapshots
	branches, err := git.ListBranches("snapshot/*")
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	var related []relatedSnapshot

	for _, branch := range branches {
		// Skip the target snapshot itself
		if branch == targetBranch {
			continue
		}

		info, err := parseSnapshotBranch(branch)
		if err != nil {
			continue
		}
		info.LoadMetadata()

		if info.Metadata == nil {
			continue
		}

		score := 0
		reasons := []string{}

		// Check explicit relationship
		if targetInfo.Metadata.RelatedTo != nil {
			for _, rel := range targetInfo.Metadata.RelatedTo {
				if rel == branch {
					score += 100
					reasons = append(reasons, "explicitly related")
					break
				}
			}
		}

		// Check reverse relationship
		if info.Metadata.RelatedTo != nil {
			for _, rel := range info.Metadata.RelatedTo {
				if rel == targetBranch {
					score += 100
					reasons = append(reasons, "explicitly related")
					break
				}
			}
		}

		// Check shared tags
		sharedTags := 0
		for _, targetTag := range targetInfo.Metadata.Tags {
			for _, tag := range info.Metadata.Tags {
				if targetTag == tag {
					sharedTags++
					break
				}
			}
		}
		if sharedTags > 0 {
			score += sharedTags * 10
			reasons = append(reasons, fmt.Sprintf("%d shared tags", sharedTags))
		}

		// Check same topic
		if info.Topic == targetInfo.Topic {
			score += 20
			reasons = append(reasons, "same topic")
		}

		// Only include if there's some relationship
		if score > 0 {
			reasonStr := ""
			for i, r := range reasons {
				if i > 0 {
					reasonStr += ", "
				}
				reasonStr += r
			}

			related = append(related, relatedSnapshot{
				Snapshot: info,
				Score:    score,
				Reason:   reasonStr,
			})
		}
	}

	if len(related) == 0 {
		fmt.Println("No related snapshots found")
		return nil
	}

	// Sort by score (highest first)
	for i := 0; i < len(related); i++ {
		for j := i + 1; j < len(related); j++ {
			if related[j].Score > related[i].Score {
				related[i], related[j] = related[j], related[i]
			}
		}
	}

	// Output JSON if requested
	if relatedJSON {
		output, err := json.MarshalIndent(related, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Output Toon if requested
	if relatedToon {
		output, err := gotoon.Encode(related)
		if err != nil {
			return fmt.Errorf("failed to encode Toon: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	// Display human-readable results
	fmt.Printf("Found %d related snapshot(s) for %s:\n\n", len(related), targetBranch)

	for i, r := range related {
		fmt.Printf("%d. %s [score: %d]\n", i+1, r.Snapshot.Branch, r.Score)
		fmt.Printf("   Relationship: %s\n", r.Reason)
		fmt.Printf("   Topic:   %s\n", r.Snapshot.Topic)
		fmt.Printf("   Created: %s\n", r.Snapshot.Timestamp.Format("2006-01-02 15:04"))

		if meta := r.Snapshot.Metadata; meta != nil {
			if len(meta.Tags) > 0 {
				fmt.Printf("   Tags:    %v\n", meta.Tags)
			}
			if meta.Notes != "" {
				notes := meta.Notes
				if len(notes) > 60 {
					notes = notes[:60] + "..."
				}
				fmt.Printf("   Notes:   %s\n", notes)
			}
		}
		fmt.Println()
	}

	return nil
}
