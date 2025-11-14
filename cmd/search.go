package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/paulderscheid/git-context/internal/git"
	"github.com/paulderscheid/git-context/internal/models"
	"github.com/spf13/cobra"
)

var (
	searchTopic string
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search snapshots by keyword",
	Long: `Search through snapshot metadata, notes, and tags using keywords.

This performs a simple keyword search. For semantic search using embeddings,
use Phase 3 functionality (not yet implemented).

Example:
  context search "security vulnerabilities"
  context search --topic parser "fragility"`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVar(&searchTopic, "topic", "", "Filter by topic")
}

func runSearch(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	query := strings.ToLower(args[0])
	queryWords := strings.Fields(query)

	// Get all snapshot branches
	branches, err := git.ListBranches("snapshot/*")
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	// Search through snapshots
	var results []searchResult
	for _, branch := range branches {
		info, err := parseSnapshotBranch(branch)
		if err != nil {
			continue
		}

		// Apply topic filter
		if searchTopic != "" && info.Topic != searchTopic {
			continue
		}

		// Read metadata
		metaPath := models.MetadataPath(info.Timestamp, info.Topic)
		metaContent, err := gitShow(branch, metaPath)
		if err != nil {
			// Skip if metadata can't be read
			continue
		}

		var metadata models.Metadata
		if err := json.Unmarshal([]byte(metaContent), &metadata); err != nil {
			continue
		}

		// Calculate relevance score
		score := calculateRelevance(queryWords, &metadata)
		if score > 0 {
			results = append(results, searchResult{
				Info:     info,
				Metadata: metadata,
				Score:    score,
			})
		}
	}

	if len(results) == 0 {
		fmt.Println("No snapshots match the search query")
		return nil
	}

	// Sort by relevance score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Display results
	fmt.Printf("Found %d matching snapshot(s):\n\n", len(results))
	for i, r := range results {
		fmt.Printf("%d. %s (score: %d)\n", i+1, r.Info.Branch, r.Score)
		fmt.Printf("   Topic:   %s\n", r.Metadata.Topic)
		fmt.Printf("   Created: %s\n", r.Metadata.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("   Mode:    %s\n", r.Metadata.Mode)

		if len(r.Metadata.Tags) > 0 {
			fmt.Printf("   Tags:    %v\n", r.Metadata.Tags)
		}

		if r.Metadata.Notes != "" {
			notes := r.Metadata.Notes
			if len(notes) > 80 {
				notes = notes[:80] + "..."
			}
			fmt.Printf("   Notes:   %s\n", notes)
		}
		fmt.Println()
	}

	return nil
}

type searchResult struct {
	Info     snapshotInfo
	Metadata models.Metadata
	Score    int
}

func calculateRelevance(queryWords []string, metadata *models.Metadata) int {
	score := 0
	searchableText := strings.ToLower(fmt.Sprintf("%s %s %s %v",
		metadata.Topic,
		metadata.Notes,
		metadata.RelatedBranch,
		metadata.Tags,
	))

	for _, word := range queryWords {
		// Count occurrences of each query word
		count := strings.Count(searchableText, strings.ToLower(word))
		score += count * 10

		// Bonus points for exact matches in topic
		if strings.Contains(strings.ToLower(metadata.Topic), word) {
			score += 50
		}

		// Bonus points for tag matches
		for _, tag := range metadata.Tags {
			if strings.Contains(strings.ToLower(tag), word) {
				score += 30
			}
		}
	}

	return score
}
