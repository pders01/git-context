package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/pders01/git-context/internal/testutil"
)

func TestMetaCommand(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create a snapshot
	createTestSnapshot(t, "meta-test", "full", []string{"important", "test"})

	// Find the snapshot branch
	branches := repo.GetBranches()
	var timestamp, topic string
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") && strings.Contains(branch, "meta-test") {
			// Parse: snapshot/YYYY-MM-DDTHHMM/topic
			parts := strings.Split(branch, "/")
			if len(parts) == 3 {
				timestamp = parts[1]
				topic = parts[2]
			}
			break
		}
	}

	if timestamp == "" {
		t.Fatal("snapshot not found")
	}

	// Run meta command
	err := runMeta(nil, []string{timestamp, topic})
	if err != nil {
		t.Fatalf("meta command failed: %v", err)
	}
}

func TestMetaInvalidTimestamp(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Invalid timestamp format
	err := runMeta(nil, []string{"invalid-timestamp", "topic"})
	if err == nil {
		t.Error("expected error with invalid timestamp")
	}
}

func TestMetaBranchNotFound(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Valid timestamp format but non-existent branch
	err := runMeta(nil, []string{"2025-01-01T1200", "nonexistent"})
	if err == nil {
		t.Error("expected error when branch doesn't exist")
	}
}
