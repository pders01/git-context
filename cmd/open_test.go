package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pders01/git-context/internal/testutil"
)

func TestOpenCommand(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create a snapshot
	createTestSnapshot(t, "open-test", "full", []string{})

	// Find the snapshot branch
	branches := repo.GetBranches()
	var timestamp, topic string
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") && strings.Contains(branch, "open-test") {
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

	// Set custom worktree path in temp dir
	worktreePath := filepath.Join(repo.Path, "..", "test-worktree")
	openPath = worktreePath

	// Run open command
	err := runOpen(nil, []string{timestamp, topic})
	if err != nil {
		t.Fatalf("open command failed: %v", err)
	}

	// Verify worktree was created
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree was not created")
	}

	// Cleanup worktree manually
	os.RemoveAll(worktreePath)
	openPath = ""
}

func TestOpenInvalidTimestamp(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	openPath = ""

	// Invalid timestamp format
	err := runOpen(nil, []string{"invalid-timestamp", "topic"})
	if err == nil {
		t.Error("expected error with invalid timestamp")
	}
}

func TestOpenBranchNotFound(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	openPath = ""

	// Valid timestamp but non-existent branch
	err := runOpen(nil, []string{"2025-01-01T1200", "nonexistent"})
	if err == nil {
		t.Error("expected error when branch doesn't exist")
	}
}
