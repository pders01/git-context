package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/pders01/git-context/internal/testutil"
)

func TestPruneNoSnapshots(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Reset flags
	pruneDryRun = true
	pruneForce = false

	// Should succeed with no snapshots
	err := runPrune(nil, []string{})
	if err != nil {
		t.Fatalf("prune command failed: %v", err)
	}
}

func TestPruneDryRun(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create some snapshots
	createTestSnapshot(t, "test1", "full", []string{})
	createTestSnapshot(t, "test2", "full", []string{})

	// Dry run should not delete anything
	pruneDryRun = true
	pruneForce = false

	err := runPrune(nil, []string{})
	if err != nil {
		t.Fatalf("prune command failed: %v", err)
	}

	// Verify snapshots still exist
	branches := repo.GetBranches()
	snapshotCount := 0
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") {
			snapshotCount++
		}
	}

	if snapshotCount != 2 {
		t.Errorf("expected 2 snapshots after dry-run, got %d", snapshotCount)
	}
}

func TestPruneWithPreserveTags(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create snapshot with preserve tag
	createTestSnapshot(t, "important-snapshot", "full", []string{"important"})
	createTestSnapshot(t, "regular-snapshot", "full", []string{})

	// Prune (dry run)
	pruneDryRun = true
	pruneForce = false

	err := runPrune(nil, []string{})
	if err != nil {
		t.Fatalf("prune command failed: %v", err)
	}

	// All snapshots should be preserved (they're recent)
	branches := repo.GetBranches()
	snapshotCount := 0
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") {
			snapshotCount++
		}
	}

	if snapshotCount != 2 {
		t.Errorf("expected 2 snapshots, got %d", snapshotCount)
	}
}
