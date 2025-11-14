package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/pders01/git-context/internal/testutil"
)

func TestListNoSnapshots(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Reset flags
	listTopic = ""
	listToday = false
	listSince = ""

	// Should succeed with no snapshots
	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}
}

func TestListWithSnapshots(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create some snapshots
	createTestSnapshot(t, "first-snapshot", "full", []string{"tag1"})
	createTestSnapshot(t, "second-snapshot", "research-only", []string{"tag2"})

	// Reset flags
	listTopic = ""
	listToday = false
	listSince = ""

	// Should list both snapshots
	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	// Verify branches exist
	branches := repo.GetBranches()
	snapshotCount := 0
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") {
			snapshotCount++
		}
	}

	if snapshotCount != 2 {
		t.Errorf("expected 2 snapshot branches, got %d", snapshotCount)
	}
}

func TestListWithTopicFilter(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create snapshots with different topics
	createTestSnapshot(t, "security-audit", "full", []string{})
	createTestSnapshot(t, "performance-test", "full", []string{})
	createTestSnapshot(t, "security-fix", "full", []string{})

	// Filter by topic
	listTopic = "security-audit"
	listToday = false
	listSince = ""

	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	// Reset filter
	listTopic = ""
}

func TestListToday(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create a snapshot (will be today)
	createTestSnapshot(t, "today-snapshot", "full", []string{})

	// List today's snapshots
	listTopic = ""
	listToday = true
	listSince = ""

	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	listToday = false
}

func TestListSince(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create a snapshot
	createTestSnapshot(t, "recent-snapshot", "full", []string{})

	// List since a date in the past
	listTopic = ""
	listToday = false
	listSince = "2025-01-01"

	err := runList(nil, []string{})
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	listSince = ""
}

func TestListInvalidSinceDate(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Invalid date format should fail
	listTopic = ""
	listToday = false
	listSince = "invalid-date"

	err := runList(nil, []string{})
	if err == nil {
		t.Error("expected error with invalid date format")
	}

	listSince = ""
}

// Helper function to create test snapshots
func createTestSnapshot(t *testing.T, topic, mode string, tags []string) {
	t.Helper()

	saveTopic = ""
	saveMode = mode
	saveTags = tags
	saveNotes = "Test snapshot"
	saveNoEmbed = true
	saveInclude = []string{}

	if mode == "poc" {
		saveInclude = []string{"README.md"}
	}

	err := runSave(nil, []string{topic})
	if err != nil {
		t.Fatalf("failed to create test snapshot: %v", err)
	}
}
