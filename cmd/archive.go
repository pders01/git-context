package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/paulderscheid/git-context/internal/git"
	"github.com/spf13/cobra"
)

var (
	archiveOutput string
	archiveTopic  string
)

var archiveCmd = &cobra.Command{
	Use:   "archive <year|YYYY-MM|all>",
	Short: "Bundle snapshots for external storage",
	Long: `Create a tar.gz archive of snapshot branches for backup or transfer.

Examples:
  context archive 2024          # Archive all snapshots from 2024
  context archive 2024-11       # Archive snapshots from November 2024
  context archive all           # Archive all snapshots
  context archive 2024 --topic security  # Archive only security snapshots from 2024
  context archive all --output my-snapshots.tar.gz`,
	Args: cobra.ExactArgs(1),
	RunE: runArchive,
}

func init() {
	rootCmd.AddCommand(archiveCmd)

	archiveCmd.Flags().StringVar(&archiveOutput, "output", "", "Output file path (default: context-snapshots-<period>.tar.gz)")
	archiveCmd.Flags().StringVar(&archiveTopic, "topic", "", "Filter by topic")
}

func runArchive(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	period := args[0]

	// Get all snapshot branches
	branches, err := git.ListBranches("snapshot/*")
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	// Filter branches by period and topic
	var selectedBranches []string
	for _, branch := range branches {
		// Parse branch: snapshot/YYYY-MM-DDTHHMM/topic
		parts := strings.Split(branch, "/")
		if len(parts) != 3 {
			continue
		}

		timestamp := parts[1]

		// Apply period filter
		if period != "all" {
			if !strings.HasPrefix(timestamp, period) {
				continue
			}
		}

		// Apply topic filter
		if archiveTopic != "" {
			topic := parts[2]
			if topic != archiveTopic {
				continue
			}
		}

		selectedBranches = append(selectedBranches, branch)
	}

	if len(selectedBranches) == 0 {
		fmt.Println("No snapshots match the filter criteria")
		return nil
	}

	// Determine output file
	outputFile := archiveOutput
	if outputFile == "" {
		if period == "all" {
			outputFile = "context-snapshots-all.tar.gz"
		} else {
			outputFile = fmt.Sprintf("context-snapshots-%s.tar.gz", strings.ReplaceAll(period, "/", "-"))
		}
	}

	fmt.Printf("Archiving %d snapshot(s) to: %s\n", len(selectedBranches), outputFile)
	fmt.Println()

	// Create archive
	if err := createArchive(outputFile, selectedBranches); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	// Get file size
	fileInfo, err := os.Stat(outputFile)
	if err == nil {
		sizeKB := fileInfo.Size() / 1024
		fmt.Printf("\n✓ Archive created: %s (%.2f KB)\n", outputFile, float64(sizeKB))
	} else {
		fmt.Printf("\n✓ Archive created: %s\n", outputFile)
	}

	fmt.Println("\nArchived snapshots:")
	for _, branch := range selectedBranches {
		fmt.Printf("  - %s\n", branch)
	}

	return nil
}

func createArchive(filename string, branches []string) error {
	// Create output file
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Create temporary directory for exporting branches
	tmpDir, err := os.MkdirTemp("", "context-archive-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Export each branch using worktree
	for i, branch := range branches {
		fmt.Printf("  [%d/%d] Exporting %s...\n", i+1, len(branches), branch)

		// Create worktree for the branch
		worktreePath := filepath.Join(tmpDir, strings.ReplaceAll(branch, "/", "-"))
		if err := git.CreateWorktree(worktreePath, branch); err != nil {
			return fmt.Errorf("failed to create worktree for %s: %w", branch, err)
		}

		// Add worktree contents to archive
		branchPrefix := strings.ReplaceAll(branch, "/", "-")
		err := filepath.Walk(worktreePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip .git directory
			if info.IsDir() && info.Name() == ".git" {
				return filepath.SkipDir
			}

			// Get relative path
			relPath, err := filepath.Rel(worktreePath, path)
			if err != nil {
				return err
			}

			// Skip root directory itself
			if relPath == "." {
				return nil
			}

			// Create tar header
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			// Set name with branch prefix
			header.Name = filepath.Join(branchPrefix, relPath)

			// Write header
			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}

			// Write file content if it's a regular file
			if info.Mode().IsRegular() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				if _, err := io.Copy(tarWriter, file); err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to archive %s: %w", branch, err)
		}

		// Remove worktree
		if err := git.RemoveWorktree(worktreePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree: %v\n", err)
		}
	}

	return nil
}
