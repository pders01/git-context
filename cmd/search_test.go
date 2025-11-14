package cmd

import (
	"os"
	"testing"

	"github.com/pders01/git-context/internal/models"
	"github.com/pders01/git-context/internal/testutil"
)

func TestSearchNoSnapshots(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	searchTopic = ""

	// Should succeed with no results
	err := runSearch(nil, []string{"test query"})
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}
}

func TestSearchWithResults(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create snapshots with searchable content
	saveTopic = ""
	saveMode = "full"
	saveTags = []string{"security"}
	saveNotes = "Found vulnerability in authentication"
	saveNoEmbed = true
	saveInclude = []string{}

	err := runSave(nil, []string{"security-audit"})
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	// Search for content that should match
	searchTopic = ""
	err = runSearch(nil, []string{"security"})
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}

	// Search for content in notes
	err = runSearch(nil, []string{"vulnerability"})
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}
}

func TestSearchWithTopicFilter(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create multiple snapshots
	createTestSnapshot(t, "security-audit", "full", []string{"security"})
	createTestSnapshot(t, "performance-test", "full", []string{"perf"})

	// Search with topic filter
	searchTopic = "security-audit"
	err := runSearch(nil, []string{"audit"})
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}

	searchTopic = ""
}

func TestSearchNoMatches(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create a snapshot
	createTestSnapshot(t, "test-snapshot", "full", []string{})

	// Search for something that won't match
	searchTopic = ""
	err := runSearch(nil, []string{"nonexistent-query-string-xyz"})
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}
}

func TestCalculateRelevance(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		metadata  *models.Metadata
		minScore  int // Minimum expected score
	}{
		{
			name:  "exact topic match",
			query: "security",
			metadata: &models.Metadata{
				Topic: "security",
				Notes: "",
				Tags:  []string{},
			},
			minScore: 50, // Bonus for topic match
		},
		{
			name:  "tag match",
			query: "bug",
			metadata: &models.Metadata{
				Topic: "test",
				Notes: "",
				Tags:  []string{"bug", "feature"},
			},
			minScore: 30, // Bonus for tag match
		},
		{
			name:  "notes match",
			query: "vulnerability",
			metadata: &models.Metadata{
				Topic: "security",
				Notes: "Found vulnerability in authentication",
				Tags:  []string{},
			},
			minScore: 10, // Word occurrence
		},
		{
			name:  "multiple word match",
			query: "security vulnerability",
			metadata: &models.Metadata{
				Topic: "security",
				Notes: "Found vulnerability in authentication",
				Tags:  []string{"security"},
			},
			minScore: 100, // Multiple matches across fields
		},
		{
			name:  "no match",
			query: "xyz",
			metadata: &models.Metadata{
				Topic: "test",
				Notes: "Some notes",
				Tags:  []string{"tag"},
			},
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedQuery := parseSearchQuery(tt.query)
			score, shouldExclude := calculateRelevance(parsedQuery, tt.metadata)

			if shouldExclude {
				t.Errorf("unexpected exclusion for query: %s", tt.query)
			}

			if score < tt.minScore {
				t.Errorf("expected score >= %d, got %d", tt.minScore, score)
			}
		})
	}
}

func TestSearchRanking(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create snapshots with different relevance levels
	// Snapshot 1: High relevance (exact topic match)
	saveTopic = ""
	saveMode = "full"
	saveTags = []string{}
	saveNotes = ""
	saveNoEmbed = true
	saveInclude = []string{}
	runSave(nil, []string{"security"})

	// Snapshot 2: Medium relevance (tag match)
	saveTags = []string{"security"}
	saveNotes = ""
	runSave(nil, []string{"feature-test"})

	// Snapshot 3: Low relevance (only in notes)
	saveTags = []string{}
	saveNotes = "Mentioned security in passing"
	runSave(nil, []string{"unrelated"})

	// Search should rank them appropriately
	searchTopic = ""
	err := runSearch(nil, []string{"security"})
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}

	// Note: We can't easily verify the order without capturing output
	// This test at least verifies the search completes successfully
}

func TestSearchKeywordOnly(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create snapshot without embeddings
	createTestSnapshot(t, "test-topic", "full", []string{"test"})

	// Search should work with keyword-only mode
	// (since Ollama is likely not available in test environment)
	searchTopic = ""
	err := runSearch(nil, []string{"test"})
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}

	// Search completes successfully even without embeddings
}

func TestSearchEmptyQuery(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	createTestSnapshot(t, "test-topic", "full", []string{})

	// Even with empty query words (after processing),
	// search should not crash
	searchTopic = ""
	err := runSearch(nil, []string{" "})
	if err != nil {
		t.Fatalf("search command failed: %v", err)
	}
}

// Helper function to split query into words (mimics search.go logic)
func splitQueryWords(query string) []string {
	words := []string{}
	for _, word := range splitWords(query) {
		if word != "" {
			words = append(words, toLowerCase(word))
		}
	}
	return words
}

func splitWords(s string) []string {
	var words []string
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

func toLowerCase(s string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result += string(c)
	}
	return result
}
