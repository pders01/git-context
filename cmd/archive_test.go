package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paulderscheid/git-context/internal/testutil"
)

func TestArchiveNoSnapshots(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Reset flags
	archiveOutput = ""
	archiveTopic = ""

	// Should succeed with no snapshots
	err := runArchive(nil, []string{"all"})
	if err != nil {
		t.Fatalf("archive command failed: %v", err)
	}
}

func TestArchiveAll(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create some snapshots
	createTestSnapshot(t, "snap1", "full", []string{})
	createTestSnapshot(t, "snap2", "research-only", []string{})

	// Set custom output
	archiveFile := filepath.Join(repo.Path, "test-archive.tar.gz")
	archiveOutput = archiveFile
	archiveTopic = ""

	err := runArchive(nil, []string{"all"})
	if err != nil {
		t.Fatalf("archive command failed: %v", err)
	}

	// Verify archive was created
	if _, err := os.Stat(archiveFile); os.IsNotExist(err) {
		t.Error("archive file was not created")
	}

	// Cleanup
	os.Remove(archiveFile)
	archiveOutput = ""
}

func TestArchiveByPeriod(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create a snapshot
	createTestSnapshot(t, "test-snap", "full", []string{})

	// Archive by year (current year)
	archiveFile := filepath.Join(repo.Path, "test-2025.tar.gz")
	archiveOutput = archiveFile
	archiveTopic = ""

	err := runArchive(nil, []string{"2025"})
	if err != nil {
		t.Fatalf("archive command failed: %v", err)
	}

	// Verify archive exists
	if _, err := os.Stat(archiveFile); os.IsNotExist(err) {
		t.Error("archive file was not created")
	}

	// Cleanup
	os.Remove(archiveFile)
	archiveOutput = ""
}

func TestArchiveByTopic(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create snapshots with different topics
	createTestSnapshot(t, "security-test", "full", []string{})
	createTestSnapshot(t, "performance-test", "full", []string{})

	// Archive only security topic
	archiveFile := filepath.Join(repo.Path, "test-security.tar.gz")
	archiveOutput = archiveFile
	archiveTopic = "security-test"

	err := runArchive(nil, []string{"all"})
	if err != nil {
		t.Fatalf("archive command failed: %v", err)
	}

	// Verify archive exists
	if _, err := os.Stat(archiveFile); os.IsNotExist(err) {
		t.Error("archive file was not created")
	}

	// Cleanup
	os.Remove(archiveFile)
	archiveOutput = ""
	archiveTopic = ""
}

func TestArchiveDefaultFilename(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create a snapshot
	createTestSnapshot(t, "test", "full", []string{})

	// Archive without specifying output (uses default filename)
	archiveOutput = ""
	archiveTopic = ""

	err := runArchive(nil, []string{"all"})
	if err != nil {
		t.Fatalf("archive command failed: %v", err)
	}

	// Check if default file was created
	defaultFile := filepath.Join(repo.Path, "context-snapshots-all.tar.gz")
	if _, err := os.Stat(defaultFile); err == nil {
		// Cleanup
		os.Remove(defaultFile)
	}
}
