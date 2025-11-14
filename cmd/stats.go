package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/alpkeskin/gotoon"
	"github.com/pders01/git-context/internal/git"
	"github.com/spf13/cobra"
)

var (
	statsJSON bool
	statsToon bool
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show snapshot statistics and analytics",
	Long: `Display statistics about your snapshots including:
  - Total snapshot count
  - Snapshots by mode (full, research-only, diff, poc)
  - Tag usage statistics
  - Timeline distribution
  - Embedding coverage

Examples:
  context stats
  context stats --json
  context stats --toon`,
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)

	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "Output as JSON")
	statsCmd.Flags().BoolVar(&statsToon, "toon", false, "Output in LLM-friendly toon format")
}

type snapshotStats struct {
	TotalSnapshots   int               `json:"total_snapshots"`
	ByMode           map[string]int    `json:"by_mode"`
	ByTag            map[string]int    `json:"by_tag"`
	ByDate           map[string]int    `json:"by_date"`
	WithEmbeddings   int               `json:"with_embeddings"`
	WithoutEmbeddings int              `json:"without_embeddings"`
	OldestSnapshot   *time.Time        `json:"oldest_snapshot,omitempty"`
	NewestSnapshot   *time.Time        `json:"newest_snapshot,omitempty"`
	TopTags          []tagStat         `json:"top_tags"`
	DailyActivity    []dailyActivity   `json:"daily_activity"`
}

type tagStat struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type dailyActivity struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

func runStats(cmd *cobra.Command, args []string) error {
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

	// Collect statistics
	stats := &snapshotStats{
		ByMode: make(map[string]int),
		ByTag:  make(map[string]int),
		ByDate: make(map[string]int),
	}

	var snapshots []snapshotInfo
	for _, branch := range branches {
		info, err := parseSnapshotBranch(branch)
		if err != nil {
			continue
		}
		info.LoadMetadata()
		snapshots = append(snapshots, info)
	}

	stats.TotalSnapshots = len(snapshots)

	// Analyze snapshots
	for _, s := range snapshots {
		// Track oldest/newest
		if stats.OldestSnapshot == nil || s.Timestamp.Before(*stats.OldestSnapshot) {
			t := s.Timestamp
			stats.OldestSnapshot = &t
		}
		if stats.NewestSnapshot == nil || s.Timestamp.After(*stats.NewestSnapshot) {
			t := s.Timestamp
			stats.NewestSnapshot = &t
		}

		// Count by mode
		if s.Metadata != nil {
			mode := string(s.Metadata.Mode)
			stats.ByMode[mode]++

			// Count tags
			for _, tag := range s.Metadata.Tags {
				stats.ByTag[tag]++
			}
		}

		// Count by date
		date := s.Timestamp.Format("2006-01-02")
		stats.ByDate[date]++

		// Count embeddings
		if s.HasEmbedding {
			stats.WithEmbeddings++
		} else {
			stats.WithoutEmbeddings++
		}
	}

	// Build top tags list
	for tag, count := range stats.ByTag {
		stats.TopTags = append(stats.TopTags, tagStat{Tag: tag, Count: count})
	}
	sort.Slice(stats.TopTags, func(i, j int) bool {
		return stats.TopTags[i].Count > stats.TopTags[j].Count
	})

	// Build daily activity
	for date, count := range stats.ByDate {
		stats.DailyActivity = append(stats.DailyActivity, dailyActivity{Date: date, Count: count})
	}
	sort.Slice(stats.DailyActivity, func(i, j int) bool {
		return stats.DailyActivity[i].Date > stats.DailyActivity[j].Date
	})

	// Output JSON if requested
	if statsJSON {
		output, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Output Toon if requested
	if statsToon {
		output, err := gotoon.Encode(stats)
		if err != nil {
			return fmt.Errorf("failed to encode Toon: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	// Display human-readable stats
	fmt.Println("Snapshot Statistics")
	fmt.Println("━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Printf("Total Snapshots: %d\n", stats.TotalSnapshots)
	if stats.OldestSnapshot != nil && stats.NewestSnapshot != nil {
		fmt.Printf("Date Range:      %s to %s\n",
			stats.OldestSnapshot.Format("2006-01-02"),
			stats.NewestSnapshot.Format("2006-01-02"))
	}
	fmt.Println()

	// Mode breakdown
	fmt.Println("By Mode:")
	for _, mode := range []string{"full", "research-only", "diff", "poc"} {
		if count, ok := stats.ByMode[mode]; ok {
			percentage := float64(count) / float64(stats.TotalSnapshots) * 100
			fmt.Printf("  %-15s %3d  (%.1f%%)\n", mode, count, percentage)
		}
	}
	fmt.Println()

	// Embedding coverage
	fmt.Println("Embedding Coverage:")
	if stats.TotalSnapshots > 0 {
		percentage := float64(stats.WithEmbeddings) / float64(stats.TotalSnapshots) * 100
		fmt.Printf("  With embeddings:    %3d  (%.1f%%)\n", stats.WithEmbeddings, percentage)
		fmt.Printf("  Without embeddings: %3d  (%.1f%%)\n", stats.WithoutEmbeddings, 100-percentage)
	}
	fmt.Println()

	// Top tags
	if len(stats.TopTags) > 0 {
		fmt.Println("Top Tags:")
		limit := 10
		if len(stats.TopTags) < limit {
			limit = len(stats.TopTags)
		}
		for i := 0; i < limit; i++ {
			ts := stats.TopTags[i]
			fmt.Printf("  %-20s %3d\n", ts.Tag, ts.Count)
		}
		fmt.Println()
	}

	// Recent activity
	if len(stats.DailyActivity) > 0 {
		fmt.Println("Recent Activity:")
		limit := 7
		if len(stats.DailyActivity) < limit {
			limit = len(stats.DailyActivity)
		}
		for i := 0; i < limit; i++ {
			da := stats.DailyActivity[i]
			bar := ""
			for j := 0; j < da.Count && j < 20; j++ {
				bar += "█"
			}
			fmt.Printf("  %s  %3d  %s\n", da.Date, da.Count, bar)
		}
	}

	return nil
}
