package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// IsGitRepo checks if current directory is a git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCurrentCommit returns the current commit hash
func GetCurrentCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetTreeHash returns the tree hash of current HEAD
func GetTreeHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD^{tree}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get tree hash: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// BranchExists checks if a branch exists
func BranchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	return cmd.Run() == nil
}

// CreateBranch creates a new branch
func CreateBranch(branch string) error {
	cmd := exec.Command("git", "branch", branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branch, err)
	}
	return nil
}

// CheckoutBranch checks out a branch
func CheckoutBranch(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branch, err)
	}
	return nil
}

// CheckoutBranchForce checks out a branch with force flag
func CheckoutBranchForce(branch string) error {
	cmd := exec.Command("git", "checkout", "-f", branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to force checkout branch %s: %w", branch, err)
	}
	return nil
}

// AddFiles stages files for commit
func AddFiles(files ...string) error {
	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}
	return nil
}

// Commit creates a commit with the given message
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// ListBranches returns all branches matching a pattern
func ListBranches(pattern string) ([]string, error) {
	cmd := exec.Command("git", "branch", "--list", pattern)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var branches []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove the * marker for current branch
		line = strings.TrimPrefix(line, "* ")
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// DeleteBranch deletes a branch
func DeleteBranch(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	cmd := exec.Command("git", "branch", flag, branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branch, err)
	}
	return nil
}

// CreateWorktree creates a git worktree
func CreateWorktree(path, branch string) error {
	cmd := exec.Command("git", "worktree", "add", path, branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %s: %w", string(output), err)
	}
	return nil
}

// RemoveWorktree removes a git worktree (with force to handle untracked files)
func RemoveWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %s: %w", string(output), err)
	}
	return nil
}

// GetDiff returns the diff between current state and a commit
func GetDiff(commit string) (string, error) {
	cmd := exec.Command("git", "diff", commit)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}
	return string(output), nil
}

// HasUncommittedChanges checks if there are uncommitted changes
func HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// RemoveAllFilesFromIndex removes all files from the git index (staging area)
func RemoveAllFilesFromIndex() error {
	cmd := exec.Command("git", "rm", "-r", "--cached", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove files from index: %w", err)
	}
	return nil
}

// RemoveUntrackedFiles removes all untracked files and directories
func RemoveUntrackedFiles() error {
	cmd := exec.Command("git", "clean", "-fd")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove untracked files: %w", err)
	}
	return nil
}

// AddFilesInDir stages files for commit in a specific directory
func AddFilesInDir(dir string, files ...string) error {
	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}
	return nil
}

// CommitInDir creates a commit with the given message in a specific directory
func CommitInDir(dir, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// CommitInDirNoVerify creates a commit bypassing hooks (used for snapshot creation)
func CommitInDirNoVerify(dir, message string) error {
	cmd := exec.Command("git", "commit", "--no-verify", "-m", message)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// RemoveAllFilesFromIndexInDir removes all files from the git index in a specific directory
func RemoveAllFilesFromIndexInDir(dir string) error {
	cmd := exec.Command("git", "rm", "-r", "--cached", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove files from index: %w", err)
	}
	return nil
}
