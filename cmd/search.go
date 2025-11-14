package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alpkeskin/gotoon"
	"github.com/pders01/git-context/internal/config"
	"github.com/pders01/git-context/internal/embeddings"
	"github.com/pders01/git-context/internal/git"
	"github.com/pders01/git-context/internal/models"
	"github.com/pders01/git-context/internal/ollama"
	"github.com/spf13/cobra"
)

var (
	searchTopic string
	searchJSON  bool
	searchToon  bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search snapshots using hybrid keyword and semantic search",
	Long: `Search through snapshot metadata, notes, and tags using hybrid search.

Combines keyword matching with semantic similarity (if embeddings available).
Automatically uses semantic search when snapshots have embeddings.

Boolean Search Operators:
  +term        Required term (must be present)
  -term        Excluded term (must NOT be present)
  "exact"      Exact phrase match
  word         Normal term (adds to score if present)

Examples:
  context search "security vulnerabilities"
  context search +ollama +embedding -deprecated
  context search "authentication bug" +security -false-positive
  context search --topic parser "fragility"

Search Modes:
  - Keyword only: When embeddings unavailable or Ollama not running
  - Hybrid: Combines keyword (30%) + semantic (70%) when embeddings available`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVar(&searchTopic, "topic", "", "Filter by topic")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
	searchCmd.Flags().BoolVar(&searchToon, "toon", false, "Output in LLM-friendly toon format")
}

type searchQuery struct {
	required []string   // +term (must include)
	excluded []string   // -term (must exclude)
	phrases  []string   // "exact phrase"
	normal   []string   // regular terms
}

func parseSearchQuery(query string) searchQuery {
	sq := searchQuery{}

	// Extract exact phrases first
	inQuote := false
	currentPhrase := ""
	remaining := ""

	for i := 0; i < len(query); i++ {
		if query[i] == '"' {
			if inQuote {
				if currentPhrase != "" {
					sq.phrases = append(sq.phrases, strings.ToLower(currentPhrase))
				}
				currentPhrase = ""
				inQuote = false
			} else {
				inQuote = true
			}
		} else if inQuote {
			currentPhrase += string(query[i])
		} else {
			remaining += string(query[i])
		}
	}

	// Parse remaining terms (with + and - prefixes)
	words := strings.Fields(remaining)
	for _, word := range words {
		if len(word) == 0 {
			continue
		}

		if word[0] == '+' {
			if len(word) > 1 {
				sq.required = append(sq.required, strings.ToLower(word[1:]))
			}
		} else if word[0] == '-' {
			if len(word) > 1 {
				sq.excluded = append(sq.excluded, strings.ToLower(word[1:]))
			}
		} else {
			sq.normal = append(sq.normal, strings.ToLower(word))
		}
	}

	return sq
}

func runSearch(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	query := args[0]
	parsedQuery := parseSearchQuery(query)

	// For embedding generation, use the full query text
	embeddingQuery := query

	// Get all snapshot branches
	branches, err := git.ListBranches("snapshot/*")
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	// Try to generate query embedding for semantic search
	var queryEmbedding []float64
	useSemanticSearch := false

	if config.GetEmbeddingsEnabled() && ollama.IsAvailable(config.GetOllamaURL()) {
		client, err := ollama.NewClient(config.GetOllamaURL(), config.GetEmbeddingModel())
		if err == nil {
			queryEmbedding, err = client.GenerateEmbedding(embeddingQuery)
			if err == nil {
				useSemanticSearch = true
				if !searchJSON && !searchToon {
					fmt.Println("Using hybrid search (keyword + semantic)")
				}
			}
		}
	}

	if !useSemanticSearch && !searchJSON && !searchToon {
		fmt.Println("Using keyword search only")
	}

	// Get weight configuration
	keywordWeight := config.GetKeywordWeight()
	semanticWeight := config.GetSemanticWeight()

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
			continue
		}

		var metadata models.Metadata
		if err := json.Unmarshal([]byte(metaContent), &metadata); err != nil {
			continue
		}

		// Calculate keyword relevance score with boolean operators
		keywordScore, shouldExclude := calculateRelevance(parsedQuery, &metadata)
		if shouldExclude {
			continue
		}

		// Try to calculate semantic similarity
		var semanticScore float64
		hasEmbedding := false
		usedSemantic := false

		if useSemanticSearch && metadata.Embedding != "" {
			// Load snapshot embedding from branch
			embeddingPath := filepath.Join(models.ResearchPath(info.Timestamp, info.Topic), metadata.Embedding)
			embeddingContent, err := gitShow(branch, embeddingPath)
			if err == nil {
				// Write to temp file to read binary
				tmpFile := filepath.Join("/tmp", fmt.Sprintf("embedding-%s-%s.bin", info.Timestamp.Format("20060102T1504"), info.Topic))
				if err := os.WriteFile(tmpFile, []byte(embeddingContent), 0644); err == nil {
					defer os.Remove(tmpFile)

					snapshotEmbedding, err := embeddings.ReadEmbedding(tmpFile)
					if err == nil {
						similarity, err := embeddings.CosineSimilarity(queryEmbedding, snapshotEmbedding)
						if err == nil {
							// Convert similarity from [-1, 1] to [0, 100] for consistency
							semanticScore = (similarity + 1) * 50
							hasEmbedding = true
							usedSemantic = true
						}
					}
				}
			}
		}

		// Calculate combined score
		var finalScore float64
		if usedSemantic {
			// Hybrid: weighted combination
			// Normalize keyword score to 0-100 range (divide by 2 for rough normalization)
			normalizedKeyword := float64(keywordScore) / 2.0
			if normalizedKeyword > 100 {
				normalizedKeyword = 100
			}
			finalScore = keywordWeight*normalizedKeyword + semanticWeight*semanticScore
		} else {
			// Keyword only
			finalScore = float64(keywordScore)
		}

		// Only include results with some relevance
		if finalScore > 0 || keywordScore > 0 {
			results = append(results, searchResult{
				Info:          info,
				Metadata:      metadata,
				Score:         finalScore,
				KeywordScore:  keywordScore,
				SemanticScore: semanticScore,
				HasEmbedding:  hasEmbedding,
				UsedSemantic:  usedSemantic,
			})
		}
	}

	if len(results) == 0 {
		fmt.Println("No snapshots match the search query")
		return nil
	}

	// Sort by combined score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Output JSON if requested
	if searchJSON {
		output, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Output Toon if requested
	if searchToon {
		output, err := gotoon.Encode(results)
		if err != nil {
			return fmt.Errorf("failed to encode Toon: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	// Display results (human-readable)
	fmt.Printf("\nFound %d matching snapshot(s):\n\n", len(results))
	for i, r := range results {
		scoreDisplay := fmt.Sprintf("%.1f", r.Score)
		if r.UsedSemantic {
			scoreDisplay += fmt.Sprintf(" (keyword: %d, semantic: %.1f%%)", r.KeywordScore, r.SemanticScore)
		} else {
			scoreDisplay += " (keyword only)"
		}

		fmt.Printf("%d. %s [score: %s]\n", i+1, r.Info.Branch, scoreDisplay)
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
	Info            snapshotInfo      `json:"info"`
	Metadata        models.Metadata   `json:"metadata"`
	Score           float64           `json:"score"`
	KeywordScore    int               `json:"keyword_score"`
	SemanticScore   float64           `json:"semantic_score"`
	HasEmbedding    bool              `json:"has_embedding"`
	UsedSemantic    bool              `json:"used_semantic"`
}

func calculateRelevance(query searchQuery, metadata *models.Metadata) (int, bool) {
	searchableText := strings.ToLower(fmt.Sprintf("%s %s %s %v",
		metadata.Topic,
		metadata.Notes,
		metadata.RelatedBranch,
		metadata.Tags,
	))

	// Check excluded terms first (must NOT contain)
	for _, excluded := range query.excluded {
		if strings.Contains(searchableText, excluded) {
			return 0, true // shouldExclude
		}
	}

	// Check required terms (must ALL be present)
	for _, required := range query.required {
		if !strings.Contains(searchableText, required) {
			return 0, true // shouldExclude
		}
	}

	// Check exact phrases (must ALL be present)
	for _, phrase := range query.phrases {
		if !strings.Contains(searchableText, phrase) {
			return 0, true // shouldExclude
		}
	}

	// Calculate score from normal and required terms
	score := 0
	allTerms := append(query.normal, query.required...)

	for _, word := range allTerms {
		// Count occurrences of each query word
		count := strings.Count(searchableText, word)
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

	// Bonus for exact phrase matches
	for _, phrase := range query.phrases {
		if strings.Contains(searchableText, phrase) {
			score += 100 // High bonus for exact phrase match
		}
		if strings.Contains(strings.ToLower(metadata.Topic), phrase) {
			score += 150 // Even higher for phrase in topic
		}
	}

	return score, false // don't exclude
}
