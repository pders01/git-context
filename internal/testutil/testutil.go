package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TempGitRepo creates a temporary git repository for testing
type TempGitRepo struct {
	Path string
	T    *testing.T
}

// NewTempGitRepo creates a new temporary git repository
func NewTempGitRepo(t *testing.T) *TempGitRepo {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "context-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	configCmds := [][]string{
		{"config", "user.name", "Test User"},
		{"config", "user.email", "test@example.com"},
	}

	for _, args := range configCmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to configure git: %v", err)
		}
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repository\n"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to add files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create initial commit: %v", err)
	}

	return &TempGitRepo{
		Path: tmpDir,
		T:    t,
	}
}

// Cleanup removes the temporary git repository
func (r *TempGitRepo) Cleanup() {
	r.T.Helper()
	if err := os.RemoveAll(r.Path); err != nil {
		r.T.Errorf("failed to cleanup temp repo: %v", err)
	}
}

// CreateFile creates a file in the repository
func (r *TempGitRepo) CreateFile(name, content string) {
	r.T.Helper()
	path := filepath.Join(r.Path, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		r.T.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		r.T.Fatalf("failed to create file: %v", err)
	}
}

// Commit stages and commits all changes
func (r *TempGitRepo) Commit(message string) {
	r.T.Helper()

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.T.Fatalf("failed to stage files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.T.Fatalf("failed to commit: %v", err)
	}
}

// GetBranches returns all branches in the repository
func (r *TempGitRepo) GetBranches() []string {
	r.T.Helper()

	cmd := exec.Command("git", "branch", "--list")
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		r.T.Fatalf("failed to list branches: %v", err)
	}

	return parseGitBranches(string(output))
}

// BranchExists checks if a branch exists
func (r *TempGitRepo) BranchExists(branch string) bool {
	r.T.Helper()

	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = r.Path
	return cmd.Run() == nil
}

// FileExists checks if a file exists in a branch
func (r *TempGitRepo) FileExists(branch, file string) bool {
	r.T.Helper()

	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", branch)
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	for _, line := range parseLines(string(output)) {
		if line == file {
			return true
		}
	}

	return false
}

// GetFileContent retrieves file content from a specific branch
func (r *TempGitRepo) GetFileContent(branch, file string) string {
	r.T.Helper()

	cmd := exec.Command("git", "show", branch+":"+file)
	cmd.Dir = r.Path
	output, err := cmd.Output()
	if err != nil {
		r.T.Fatalf("failed to read file from branch: %v", err)
	}

	return string(output)
}

// parseGitBranches parses git branch output
func parseGitBranches(output string) []string {
	var branches []string
	for _, line := range parseLines(output) {
		// Remove leading * and spaces
		branch := line
		if len(branch) > 0 && branch[0] == '*' {
			branch = branch[1:]
		}
		branch = filepath.Clean(branch)
		if branch != "" {
			branches = append(branches, branch)
		}
	}
	return branches
}

// parseLines splits output into non-empty lines
func parseLines(output string) []string {
	var lines []string
	for _, line := range splitLines(output) {
		line = trimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// trimSpace trims whitespace from a string
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
