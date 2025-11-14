package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pders01/git-context/internal/config"
	"github.com/pders01/git-context/internal/embeddings"
	"github.com/pders01/git-context/internal/git"
	"github.com/pders01/git-context/internal/models"
	"github.com/pders01/git-context/internal/ollama"
	"github.com/spf13/cobra"
)

var (
	saveTopic      string
	saveMode       string
	saveInclude    []string
	saveTags       []string
	saveNoEmbed    bool
	saveNotes      string
)

var saveCmd = &cobra.Command{
	Use:   "save [topic]",
	Short: "Create a new context snapshot",
	Long: `Create an immutable snapshot capturing the current codebase state and research artifacts.

The snapshot will be created on a new branch with timestamp and topic:
  snapshot/YYYY-MM-DDTHHMM/topic-slug

Modes:
  full (default)    - Full code tree + research artifacts
  research-only     - Only research/ + reference commit hash
  diff              - Store patch + research/ + reference commit
  poc               - Only POC files + reference commit`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSave,
}

func init() {
	rootCmd.AddCommand(saveCmd)

	saveCmd.Flags().StringVar(&saveTopic, "topic", "", "Override topic slug")
	saveCmd.Flags().StringVar(&saveMode, "mode", "", "Snapshot mode: full|research-only|diff|poc")
	saveCmd.Flags().StringSliceVar(&saveInclude, "include", []string{}, "Extra files to include")
	saveCmd.Flags().StringSliceVar(&saveTags, "tag", []string{}, "Add metadata tags")
	saveCmd.Flags().BoolVar(&saveNoEmbed, "no-embed", false, "Skip embedding generation")
	saveCmd.Flags().StringVar(&saveNotes, "notes", "", "Optional notes")
}

func runSave(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repository
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Get or generate topic
	topic := saveTopic
	if topic == "" && len(args) > 0 {
		topic = slugify(args[0])
	}
	if topic == "" {
		return fmt.Errorf("topic is required (provide as argument or use --topic)")
	}

	// Get mode
	mode := models.SnapshotMode(saveMode)
	if mode == "" {
		mode = config.GetDefaultMode()
	}

	// Validate mode
	if !isValidMode(mode) {
		return fmt.Errorf("invalid mode: %s (must be: full, research-only, diff, poc)", mode)
	}

	// Get current state BEFORE creating anything
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	currentCommit, err := git.GetCurrentCommit()
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}

	treeHash, err := git.GetTreeHash()
	if err != nil {
		return fmt.Errorf("failed to get tree hash: %w", err)
	}

	// Create snapshot
	timestamp := time.Now()
	snapshotBranch := models.BranchName(timestamp, topic)

	// Check if branch already exists
	if git.BranchExists(snapshotBranch) {
		return fmt.Errorf("snapshot branch already exists: %s (snapshots are immutable)", snapshotBranch)
	}

	fmt.Printf("Creating snapshot: %s\n", snapshotBranch)
	fmt.Printf("Mode: %s\n", mode)
	fmt.Printf("From: %s @ %s\n", currentBranch, currentCommit[:8])

	// Create snapshot branch (but don't checkout)
	if err := git.CreateBranch(snapshotBranch); err != nil {
		return err
	}

	// Create temporary worktree for snapshot
	worktreePath := filepath.Join(os.TempDir(), fmt.Sprintf("context-snapshot-%d", timestamp.Unix()))
	fmt.Printf("Creating worktree: %s\n", worktreePath)

	if err := git.CreateWorktree(worktreePath, snapshotBranch); err != nil {
		return err
	}

	// Ensure we clean up the worktree
	defer func() {
		if err := git.RemoveWorktree(worktreePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree: %v\n", err)
		}
	}()

	// Create research directory in worktree
	researchPath := models.ResearchPath(timestamp, topic)
	worktreeResearchPath := filepath.Join(worktreePath, researchPath)
	if err := os.MkdirAll(worktreeResearchPath, 0755); err != nil {
		return fmt.Errorf("failed to create research directory: %w", err)
	}

	// Create notes.md
	notesPath := filepath.Join(worktreeResearchPath, "notes.md")
	notesContent := fmt.Sprintf("# %s\n\nCreated: %s\nBranch: %s\nCommit: %s\n\n## Notes\n\n%s\n",
		topic, timestamp.Format(time.RFC3339), currentBranch, currentCommit, saveNotes)
	if err := os.WriteFile(notesPath, []byte(notesContent), 0644); err != nil {
		return fmt.Errorf("failed to create notes.md: %w", err)
	}

	// Create metadata
	metadata := &models.Metadata{
		CreatedAt:     timestamp,
		Topic:         topic,
		Root:          snapshotBranch,
		Mode:          mode,
		RelatedBranch: currentBranch,
		MainCommit:    currentCommit,
		Tags:          saveTags,
		Notes:         saveNotes,
		TreeHash:      treeHash,
	}

	// Save metadata
	metaPath := filepath.Join(worktreeResearchPath, "meta.json")
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Generate embeddings if enabled
	if !saveNoEmbed && config.GetEmbeddingsEnabled() {
		if err := generateEmbedding(metadata, worktreeResearchPath, notesContent); err != nil {
			// Don't fail the snapshot, just warn
			fmt.Fprintf(os.Stderr, "Warning: failed to generate embedding: %v\n", err)
			fmt.Fprintln(os.Stderr, "Tip: Ensure Ollama is running and the model is available: ollama pull nomic-embed-text")
		}
	}

	// Handle different modes in the worktree
	switch mode {
	case models.ModeFull:
		// Full snapshot - everything is already there
		// Just add research directory
		if err := git.AddFilesInDir(worktreePath, researchPath); err != nil {
			return err
		}

	case models.ModeResearchOnly:
		// Research only - remove everything except research/
		fmt.Println("  Removing code files (research-only mode)...")
		if err := git.RemoveAllFilesFromIndexInDir(worktreePath); err != nil {
			return err
		}
		if err := git.AddFilesInDir(worktreePath, researchPath); err != nil {
			return err
		}

	case models.ModeDiff:
		// Diff mode - create patch file
		diff, err := git.GetDiff(currentCommit)
		if err != nil {
			return fmt.Errorf("failed to get diff: %w", err)
		}
		patchPath := filepath.Join(worktreeResearchPath, "changes.patch")
		if err := os.WriteFile(patchPath, []byte(diff), 0644); err != nil {
			return fmt.Errorf("failed to write patch: %w", err)
		}
		fmt.Println("  Removing code files (diff mode - patch only)...")
		if err := git.RemoveAllFilesFromIndexInDir(worktreePath); err != nil {
			return err
		}
		if err := git.AddFilesInDir(worktreePath, researchPath); err != nil {
			return err
		}

	case models.ModePOC:
		// POC mode - only include specified files
		if len(saveInclude) == 0 {
			return fmt.Errorf("poc mode requires --include flag to specify files")
		}
		fmt.Println("  Removing code files (poc mode - selective inclusion)...")
		if err := git.RemoveAllFilesFromIndexInDir(worktreePath); err != nil {
			return err
		}
		if err := git.AddFilesInDir(worktreePath, researchPath); err != nil {
			return err
		}
		if err := git.AddFilesInDir(worktreePath, saveInclude...); err != nil {
			return err
		}
	}

	// Commit the snapshot in the worktree
	commitMsg := fmt.Sprintf("snapshot: %s\n\nMode: %s\nFrom: %s @ %s\nTags: %v",
		topic, mode, currentBranch, currentCommit[:8], saveTags)
	if err := git.CommitInDir(worktreePath, commitMsg); err != nil {
		return fmt.Errorf("failed to commit snapshot: %w", err)
	}

	fmt.Printf("\n✓ Snapshot created: %s\n", snapshotBranch)
	fmt.Printf("  Research: %s\n", researchPath)
	fmt.Printf("  Metadata: %s\n", models.MetadataPath(timestamp, topic))

	return nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove non-alphanumeric except hyphens
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func isValidMode(mode models.SnapshotMode) bool {
	switch mode {
	case models.ModeFull, models.ModeResearchOnly, models.ModeDiff, models.ModePOC:
		return true
	default:
		return false
	}
}

// generateEmbedding generates and stores an embedding for a snapshot
func generateEmbedding(metadata *models.Metadata, researchPath, notesContent string) error {
	// Check if Ollama is available
	ollamaURL := config.GetOllamaURL()
	if !ollama.IsAvailable(ollamaURL) {
		return fmt.Errorf("Ollama is not available at %s", ollamaURL)
	}

	fmt.Println("  Generating embedding...")

	// Create Ollama client
	model := config.GetEmbeddingModel()
	client, err := ollama.NewClient(ollamaURL, model)
	if err != nil {
		return fmt.Errorf("failed to create Ollama client: %w", err)
	}

	// Check if model is available
	if err := client.CheckModel(); err != nil {
		return err
	}

	// Build text to embed: notes.md content + metadata
	embeddingText := buildEmbeddingText(metadata, notesContent)

	// Truncate if too long (nomic-embed-text supports ~8K tokens, roughly 32K chars)
	maxChars := 30000
	if len(embeddingText) > maxChars {
		embeddingText = embeddingText[:maxChars]
		fmt.Printf("  Note: Truncated text to %d characters for embedding\n", maxChars)
	}

	// Generate embedding
	vec, err := client.GenerateEmbedding(embeddingText)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Validate embedding
	if err := embeddings.ValidateEmbedding(vec); err != nil {
		return fmt.Errorf("invalid embedding: %w", err)
	}

	// Write embedding to file
	embeddingPath := filepath.Join(researchPath, "embedding.bin")
	if err := embeddings.WriteEmbedding(embeddingPath, vec); err != nil {
		return fmt.Errorf("failed to write embedding: %w", err)
	}

	// Update metadata to reference embedding
	metadata.Embedding = "embedding.bin"

	// Re-write metadata with embedding reference
	metaPath := filepath.Join(researchPath, "meta.json")
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	fmt.Printf("  ✓ Embedding generated (%d dimensions)\n", len(vec))

	return nil
}

// buildEmbeddingText constructs the text to be embedded from metadata and notes
func buildEmbeddingText(metadata *models.Metadata, notesContent string) string {
	// Combine topic, tags, notes field, and notes.md content
	var parts []string

	// Add topic
	parts = append(parts, "Topic: "+metadata.Topic)

	// Add tags
	if len(metadata.Tags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(metadata.Tags, ", "))
	}

	// Add notes from metadata
	if metadata.Notes != "" {
		parts = append(parts, "Notes: "+metadata.Notes)
	}

	// Add full notes.md content
	parts = append(parts, notesContent)

	return strings.Join(parts, "\n\n")
}
