package cmd

import (
	"os"
	"testing"

	"github.com/paulderscheid/git-context/internal/testutil"
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
