package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/alpkeskin/gotoon"
	"github.com/pders01/git-context/internal/git"
	"github.com/spf13/cobra"
)

var (
	tagsJSON   bool
	tagsToon   bool
	tagsRename string
)

var tagsCmd = &cobra.Command{
	Use:   "tags [old-tag]",
	Short: "List or manage tags",
	Long: `List all tags used across snapshots with usage counts.
Optionally rename tags across all snapshots.

Examples:
  context tags                    # List all tags
  context tags --rename new-name  # Rename tag (requires tag argument)
  context tags security --rename important-security`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTags,
}

func init() {
	rootCmd.AddCommand(tagsCmd)

	tagsCmd.Flags().BoolVar(&tagsJSON, "json", false, "Output as JSON")
	tagsCmd.Flags().BoolVar(&tagsToon, "toon", false, "Output in LLM-friendly toon format")
	tagsCmd.Flags().StringVar(&tagsRename, "rename", "", "Rename tag to new value")
}

type tagInfo struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func runTags(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Handle rename mode
	if tagsRename != "" {
		if len(args) == 0 {
			return fmt.Errorf("tag name required for --rename")
		}
		return renameTag(args[0], tagsRename)
	}

	// List mode
	branches, err := git.ListBranches("snapshot/*")
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	// Collect tags
	tagCounts := make(map[string]int)

	for _, branch := range branches {
		info, err := parseSnapshotBranch(branch)
		if err != nil {
			continue
		}
		info.LoadMetadata()

		if info.Metadata != nil {
			for _, tag := range info.Metadata.Tags {
				tagCounts[tag]++
			}
		}
	}

	if len(tagCounts) == 0 {
		fmt.Println("No tags found")
		return nil
	}

	// Build sorted list
	var tags []tagInfo
	for tag, count := range tagCounts {
		tags = append(tags, tagInfo{Tag: tag, Count: count})
	}
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Count == tags[j].Count {
			return tags[i].Tag < tags[j].Tag
		}
		return tags[i].Count > tags[j].Count
	})

	// Output JSON if requested
	if tagsJSON {
		output, err := json.MarshalIndent(tags, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Output Toon if requested
	if tagsToon {
		output, err := gotoon.Encode(tags)
		if err != nil {
			return fmt.Errorf("failed to encode Toon: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	// Human-readable output
	fmt.Printf("Found %d tag(s):\n\n", len(tags))
	for _, t := range tags {
		fmt.Printf("  %-30s %3d\n", t.Tag, t.Count)
	}

	return nil
}

func renameTag(oldTag, newTag string) error {
	branches, err := git.ListBranches("snapshot/*")
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	updated := 0

	for _, branch := range branches {
		info, err := parseSnapshotBranch(branch)
		if err != nil {
			continue
		}
		info.LoadMetadata()

		if info.Metadata == nil {
			continue
		}

		// Check if this snapshot has the old tag
		hasTag := false
		for i, tag := range info.Metadata.Tags {
			if tag == oldTag {
				info.Metadata.Tags[i] = newTag
				hasTag = true
			}
		}

		if !hasTag {
			continue
		}

		// Update metadata file using git operations
		metaPath := fmt.Sprintf("research/%s/%s/meta.json",
			info.Timestamp.Format("2006-01-02T1504"), info.Topic)

		// Marshal updated metadata
		metaBytes, err := json.MarshalIndent(info.Metadata, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to marshal metadata for %s: %v\n", branch, err)
			continue
		}

		// Create temporary worktree
		tmpDir := fmt.Sprintf("/tmp/context-tag-rename-%d", info.Timestamp.Unix())
		if err := git.CreateWorktree(tmpDir, branch); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create worktree for %s: %v\n", branch, err)
			continue
		}

		// Update file
		fullPath := fmt.Sprintf("%s/%s", tmpDir, metaPath)
		if err := os.WriteFile(fullPath, metaBytes, 0644); err != nil {
			git.RemoveWorktree(tmpDir)
			fmt.Fprintf(os.Stderr, "Warning: failed to write metadata for %s: %v\n", branch, err)
			continue
		}

		// Commit change
		if err := git.AddFilesInDir(tmpDir, metaPath); err != nil {
			git.RemoveWorktree(tmpDir)
			fmt.Fprintf(os.Stderr, "Warning: failed to add file for %s: %v\n", branch, err)
			continue
		}

		commitMsg := fmt.Sprintf("Rename tag: %s → %s", oldTag, newTag)
		if err := git.CommitInDirNoVerify(tmpDir, commitMsg); err != nil {
			git.RemoveWorktree(tmpDir)
			fmt.Fprintf(os.Stderr, "Warning: failed to commit for %s: %v\n", branch, err)
			continue
		}

		// Cleanup
		git.RemoveWorktree(tmpDir)
		updated++
	}

	fmt.Printf("Renamed tag '%s' → '%s' in %d snapshot(s)\n", oldTag, newTag, updated)
	return nil
}
