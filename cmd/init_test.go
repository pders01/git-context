package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pders01/git-context/internal/testutil"
)

func TestInitCommand(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Run init command
	err := runInit(nil, []string{})
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify pre-commit hook was created
	hookPath := filepath.Join(repo.Path, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Error("pre-commit hook was not created")
	}

	// Verify hook is executable
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("failed to stat hook: %v", err)
	}

	if info.Mode()&0111 == 0 {
		t.Error("pre-commit hook is not executable")
	}
}

func TestInitWithExistingHook(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create an existing hook
	hookPath := filepath.Join(repo.Path, ".git", "hooks", "pre-commit")
	err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho 'existing hook'\n"), 0755)
	if err != nil {
		t.Fatalf("failed to create existing hook: %v", err)
	}

	// Run init command
	err = runInit(nil, []string{})
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify existing hook was preserved (not overwritten)
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook: %v", err)
	}

	hookContent := string(content)
	if hookContent != "#!/bin/sh\necho 'existing hook'\n" {
		t.Error("existing hook was overwritten")
	}
}

func TestInitNotGitRepo(t *testing.T) {
	// Create a temp dir that's NOT a git repo
	tmpDir, err := os.MkdirTemp("", "not-git-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Should fail when not in a git repo
	err = runInit(nil, []string{})
	if err == nil {
		t.Error("expected error when not in git repo")
	}
}
