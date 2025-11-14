package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/alpkeskin/gotoon"
	"github.com/pders01/git-context/internal/git"
	"github.com/pders01/git-context/internal/models"
	"github.com/spf13/cobra"
)

var (
	listTopic   string
	listToday   bool
	listSince   string
	listJSON    bool
	listToon    bool
	listTags    []string
	listGroupBy string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all context snapshots",
	Long: `List all context snapshots with optional filtering and grouping.

Examples:
  context list
  context list --topic security
  context list --tag important
  context list --tag security --tag important  # AND: both tags required
  context list --today
  context list --since 2025-10-01
  context list --group-by tag
  context list --group-by date`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listTopic, "topic", "", "Filter by topic")
	listCmd.Flags().StringSliceVar(&listTags, "tag", []string{}, "Filter by tag(s) - multiple tags use AND logic")
	listCmd.Flags().BoolVar(&listToday, "today", false, "Show only today's snapshots")
	listCmd.Flags().StringVar(&listSince, "since", "", "Show snapshots since date (YYYY-MM-DD)")
	listCmd.Flags().StringVar(&listGroupBy, "group-by", "", "Group output by: tag, date, or mode")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	listCmd.Flags().BoolVar(&listToon, "toon", false, "Output in LLM-friendly toon format")
}

func runList(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Validate --since date format early
	if listSince != "" {
		_, err := time.Parse("2006-01-02", listSince)
		if err != nil {
			return fmt.Errorf("invalid --since date format (use YYYY-MM-DD): %w", err)
		}
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

	// Load metadata for all snapshots
	for i := range snapshots {
		snapshots[i].LoadMetadata()
	}

	// Apply tag filter (needs metadata loaded) - AND logic for multiple tags
	if len(listTags) > 0 {
		var filtered []snapshotInfo
		for _, s := range snapshots {
			if s.Metadata == nil {
				continue
			}

			// Check if snapshot has ALL required tags (AND logic)
			hasAllTags := true
			for _, requiredTag := range listTags {
				found := false
				for _, snapshotTag := range s.Metadata.Tags {
					if snapshotTag == requiredTag {
						found = true
						break
					}
				}
				if !found {
					hasAllTags = false
					break
				}
			}

			if hasAllTags {
				filtered = append(filtered, s)
			}
		}
		snapshots = filtered
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots match the filter criteria")
		return nil
	}

	// Handle grouping if requested
	if listGroupBy != "" && !listJSON && !listToon {
		return displayGrouped(snapshots, listGroupBy)
	}

	// Output JSON if requested
	if listJSON {
		output, err := json.MarshalIndent(snapshots, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Output Toon if requested
	if listToon {
		output, err := gotoon.Encode(snapshots)
		if err != nil {
			return fmt.Errorf("failed to encode Toon: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	// Display snapshots (human-readable)
	fmt.Printf("Found %d snapshot(s):\n\n", len(snapshots))
	for _, s := range snapshots {
		fmt.Printf("  %s\n", s.Branch)
		fmt.Printf("    Topic:   %s\n", s.Topic)
		fmt.Printf("    Created: %s\n", s.Timestamp.Format("2006-01-02 15:04"))

		// Show metadata if available
		if meta := s.Metadata; meta != nil {
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
			// Show embedding status
			if s.HasEmbedding {
				fmt.Printf("    Embedding: ✓\n")
			}
		}
		fmt.Println()
	}

	return nil
}

// displayGrouped shows snapshots grouped by a specified field
func displayGrouped(snapshots []snapshotInfo, groupBy string) error {
	switch groupBy {
	case "tag":
		return displayGroupedByTag(snapshots)
	case "date":
		return displayGroupedByDate(snapshots)
	case "mode":
		return displayGroupedByMode(snapshots)
	default:
		return fmt.Errorf("invalid group-by value: %s (must be: tag, date, or mode)", groupBy)
	}
}

// displayGroupedByTag groups snapshots by their tags
func displayGroupedByTag(snapshots []snapshotInfo) error {
	// Collect all tags and group snapshots
	tagMap := make(map[string][]snapshotInfo)
	untagged := []snapshotInfo{}

	for _, s := range snapshots {
		if s.Metadata == nil || len(s.Metadata.Tags) == 0 {
			untagged = append(untagged, s)
			continue
		}
		for _, tag := range s.Metadata.Tags {
			tagMap[tag] = append(tagMap[tag], s)
		}
	}

	// Sort tags alphabetically
	var tags []string
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	// Display groups
	fmt.Printf("Found %d snapshot(s) grouped by tag:\n\n", len(snapshots))

	for _, tag := range tags {
		fmt.Printf("━━━ %s (%d) ━━━\n\n", tag, len(tagMap[tag]))
		for _, s := range tagMap[tag] {
			displaySnapshot(s, "  ")
		}
	}

	if len(untagged) > 0 {
		fmt.Printf("━━━ (untagged) (%d) ━━━\n\n", len(untagged))
		for _, s := range untagged {
			displaySnapshot(s, "  ")
		}
	}

	return nil
}

// displayGroupedByDate groups snapshots by date
func displayGroupedByDate(snapshots []snapshotInfo) error {
	dateMap := make(map[string][]snapshotInfo)

	for _, s := range snapshots {
		date := s.Timestamp.Format("2006-01-02")
		dateMap[date] = append(dateMap[date], s)
	}

	// Sort dates
	var dates []string
	for date := range dateMap {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	// Display groups
	fmt.Printf("Found %d snapshot(s) grouped by date:\n\n", len(snapshots))

	for _, date := range dates {
		fmt.Printf("━━━ %s (%d) ━━━\n\n", date, len(dateMap[date]))
		for _, s := range dateMap[date] {
			displaySnapshot(s, "  ")
		}
	}

	return nil
}

// displayGroupedByMode groups snapshots by mode
func displayGroupedByMode(snapshots []snapshotInfo) error {
	modeMap := make(map[string][]snapshotInfo)

	for _, s := range snapshots {
		mode := "unknown"
		if s.Metadata != nil {
			mode = string(s.Metadata.Mode)
		}
		modeMap[mode] = append(modeMap[mode], s)
	}

	// Sort modes
	var modes []string
	for mode := range modeMap {
		modes = append(modes, mode)
	}
	sort.Strings(modes)

	// Display groups
	fmt.Printf("Found %d snapshot(s) grouped by mode:\n\n", len(snapshots))

	for _, mode := range modes {
		fmt.Printf("━━━ %s (%d) ━━━\n\n", mode, len(modeMap[mode]))
		for _, s := range modeMap[mode] {
			displaySnapshot(s, "  ")
		}
	}

	return nil
}

// displaySnapshot shows a single snapshot with indentation
func displaySnapshot(s snapshotInfo, indent string) {
	fmt.Printf("%s%s\n", indent, s.Branch)
	fmt.Printf("%s  Topic:   %s\n", indent, s.Topic)
	fmt.Printf("%s  Created: %s\n", indent, s.Timestamp.Format("2006-01-02 15:04"))

	if meta := s.Metadata; meta != nil {
		fmt.Printf("%s  Mode:    %s\n", indent, meta.Mode)
		if len(meta.Tags) > 0 {
			fmt.Printf("%s  Tags:    %v\n", indent, meta.Tags)
		}
		if meta.Notes != "" {
			notes := meta.Notes
			if len(notes) > 60 {
				notes = notes[:60] + "..."
			}
			fmt.Printf("%s  Notes:   %s\n", indent, notes)
		}
		if s.HasEmbedding {
			fmt.Printf("%s  Embedding: ✓\n", indent)
		}
	}
	fmt.Println()
}

type snapshotInfo struct {
	Branch      string             `json:"branch"`
	Timestamp   time.Time          `json:"timestamp"`
	Topic       string             `json:"topic"`
	Metadata    *models.Metadata   `json:"metadata,omitempty"`
	HasEmbedding bool              `json:"has_embedding"`
}

func (s *snapshotInfo) LoadMetadata() *models.Metadata {
	if s.Metadata != nil {
		return s.Metadata
	}

	metaPath := models.MetadataPath(s.Timestamp, s.Topic)
	metaContent, err := gitShow(s.Branch, metaPath)
	if err != nil {
		return nil
	}

	var metadata models.Metadata
	if err := json.Unmarshal([]byte(metaContent), &metadata); err != nil {
		return nil
	}

	s.Metadata = &metadata

	// Check if snapshot has embedding
	if metadata.Embedding != "" {
		embeddingPath := models.ResearchPath(s.Timestamp, s.Topic) + "/" + metadata.Embedding
		_, err := gitShow(s.Branch, embeddingPath)
		s.HasEmbedding = err == nil
	}

	return &metadata
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
