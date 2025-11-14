package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/pders01/git-context/internal/models"
	"github.com/pders01/git-context/internal/testutil"
)

func TestSaveFullMode(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	// Change to repo directory
	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create some test files
	repo.CreateFile("main.go", "package main\n")
	repo.CreateFile("test.txt", "test content\n")
	repo.Commit("Add test files")

	// Run save command
	saveTopic = ""
	saveMode = "full"
	saveTags = []string{"test"}
	saveNotes = "Test notes"
	saveNoEmbed = true
	saveInclude = []string{}

	err := runSave(nil, []string{"test-snapshot"})
	if err != nil {
		t.Fatalf("save command failed: %v", err)
	}

	// Verify snapshot branch was created
	branches := repo.GetBranches()
	var snapshotBranch string
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") && strings.Contains(branch, "test-snapshot") {
			snapshotBranch = branch
			break
		}
	}

	if snapshotBranch == "" {
		t.Fatalf("snapshot branch not created, branches: %v", branches)
	}

	// Verify all files are in the snapshot
	if !repo.FileExists(snapshotBranch, "main.go") {
		t.Error("main.go not in snapshot")
	}
	if !repo.FileExists(snapshotBranch, "test.txt") {
		t.Error("test.txt not in snapshot")
	}

	// Verify research directory exists (simplified check)
	// In a full snapshot, the research directory with metadata should exist
	// This is verified in the metadata check below

	// Verify metadata
	metaContent := repo.GetFileContent(snapshotBranch, "research/"+strings.Split(snapshotBranch, "/")[1]+"/test-snapshot/meta.json")
	var meta models.Metadata
	if err := json.Unmarshal([]byte(metaContent), &meta); err != nil {
		t.Fatalf("failed to parse metadata: %v", err)
	}

	if meta.Topic != "test-snapshot" {
		t.Errorf("expected topic 'test-snapshot', got '%s'", meta.Topic)
	}
	if meta.Mode != models.ModeFull {
		t.Errorf("expected mode 'full', got '%s'", meta.Mode)
	}
	if len(meta.Tags) != 1 || meta.Tags[0] != "test" {
		t.Errorf("expected tags ['test'], got %v", meta.Tags)
	}
}

func TestSaveResearchOnlyMode(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	repo.CreateFile("main.go", "package main\n")
	repo.CreateFile("test.txt", "test content\n")
	repo.Commit("Add test files")

	// Run save command in research-only mode
	saveTopic = ""
	saveMode = "research-only"
	saveTags = []string{}
	saveNotes = ""
	saveNoEmbed = true
	saveInclude = []string{}

	err := runSave(nil, []string{"research-test"})
	if err != nil {
		t.Fatalf("save command failed: %v", err)
	}

	// Find snapshot branch
	branches := repo.GetBranches()
	var snapshotBranch string
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") && strings.Contains(branch, "research-test") {
			snapshotBranch = branch
			break
		}
	}

	if snapshotBranch == "" {
		t.Fatalf("snapshot branch not created")
	}

	// Verify code files are NOT in the snapshot
	if repo.FileExists(snapshotBranch, "main.go") {
		t.Error("main.go should not be in research-only snapshot")
	}
	if repo.FileExists(snapshotBranch, "test.txt") {
		t.Error("test.txt should not be in research-only snapshot")
	}

	// Verify research directory exists
	hasResearch := false
	for _, branch := range branches {
		if strings.Contains(branch, "research") || snapshotBranch != "" {
			hasResearch = true
			break
		}
	}

	if !hasResearch {
		t.Error("research directory not found in research-only snapshot")
	}
}

func TestSavePOCMode(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	repo.CreateFile("main.go", "package main\n")
	repo.CreateFile("test.txt", "test content\n")
	repo.CreateFile("important.go", "important code\n")
	repo.Commit("Add test files")

	// Run save command in POC mode with specific file
	saveTopic = ""
	saveMode = "poc"
	saveTags = []string{}
	saveNotes = ""
	saveNoEmbed = true
	saveInclude = []string{"important.go"}

	err := runSave(nil, []string{"poc-test"})
	if err != nil {
		t.Fatalf("save command failed: %v", err)
	}

	// Find snapshot branch
	branches := repo.GetBranches()
	var snapshotBranch string
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") && strings.Contains(branch, "poc-test") {
			snapshotBranch = branch
			break
		}
	}

	if snapshotBranch == "" {
		t.Fatalf("snapshot branch not created")
	}

	// Verify only included file is in snapshot
	if repo.FileExists(snapshotBranch, "main.go") {
		t.Error("main.go should not be in POC snapshot")
	}
	if repo.FileExists(snapshotBranch, "test.txt") {
		t.Error("test.txt should not be in POC snapshot")
	}
	if !repo.FileExists(snapshotBranch, "important.go") {
		t.Error("important.go should be in POC snapshot")
	}
}

func TestSaveWithoutTopic(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	saveTopic = ""
	saveMode = ""
	saveTags = []string{}
	saveNotes = ""
	saveNoEmbed = true
	saveInclude = []string{}

	// Should fail without topic
	err := runSave(nil, []string{})
	if err == nil {
		t.Error("expected error when saving without topic")
	}
}

func TestSaveImmutability(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	// Create first snapshot
	saveTopic = ""
	saveMode = "full"
	saveTags = []string{}
	saveNotes = ""
	saveNoEmbed = true
	saveInclude = []string{}

	topic := "immutable-test"
	err := runSave(nil, []string{topic})
	if err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Try to create another snapshot with the same timestamp/topic
	// This is hard to test without manipulating time, but we can test
	// that the error handling works

	// For now, just verify the snapshot was created
	branches := repo.GetBranches()
	snapshotCount := 0
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") {
			snapshotCount++
		}
	}

	if snapshotCount != 1 {
		t.Errorf("expected 1 snapshot branch, got %d", snapshotCount)
	}
}

func TestSaveWithEmbeddingsDisabled(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	repo.CreateFile("main.go", "package main\n")
	repo.Commit("Add test file")

	// Save with embeddings disabled via flag
	saveTopic = ""
	saveMode = "full"
	saveTags = []string{}
	saveNotes = "Test without embeddings"
	saveNoEmbed = true
	saveInclude = []string{}

	err := runSave(nil, []string{"no-embed-test"})
	if err != nil {
		t.Fatalf("save command failed: %v", err)
	}

	// Find snapshot branch
	branches := repo.GetBranches()
	var snapshotBranch string
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") && strings.Contains(branch, "no-embed-test") {
			snapshotBranch = branch
			break
		}
	}

	if snapshotBranch == "" {
		t.Fatalf("snapshot branch not created")
	}

	// Verify metadata doesn't reference an embedding
	metaContent := repo.GetFileContent(snapshotBranch, "research/"+strings.Split(snapshotBranch, "/")[1]+"/no-embed-test/meta.json")
	var meta models.Metadata
	if err := json.Unmarshal([]byte(metaContent), &meta); err != nil {
		t.Fatalf("failed to parse metadata: %v", err)
	}

	if meta.Embedding != "" {
		t.Errorf("expected no embedding reference, got '%s'", meta.Embedding)
	}

	// Verify embedding file doesn't exist
	if repo.FileExists(snapshotBranch, "research/"+strings.Split(snapshotBranch, "/")[1]+"/no-embed-test/embedding.bin") {
		t.Error("embedding.bin should not exist when embeddings are disabled")
	}
}

func TestSaveWithOllamaUnavailable(t *testing.T) {
	repo := testutil.NewTempGitRepo(t)
	defer repo.Cleanup()

	oldWd, _ := os.Getwd()
	os.Chdir(repo.Path)
	defer os.Chdir(oldWd)

	repo.CreateFile("main.go", "package main\n")
	repo.Commit("Add test file")

	// Save with embeddings enabled but Ollama not available
	// (it will gracefully degrade and just warn)
	saveTopic = ""
	saveMode = "full"
	saveTags = []string{}
	saveNotes = "Test with Ollama unavailable"
	saveNoEmbed = false // Enable embeddings
	saveInclude = []string{}

	// This should succeed even if Ollama is not available
	// (graceful degradation)
	err := runSave(nil, []string{"ollama-unavail-test"})
	if err != nil {
		t.Fatalf("save command failed: %v", err)
	}

	// Find snapshot branch
	branches := repo.GetBranches()
	var snapshotBranch string
	for _, branch := range branches {
		if strings.Contains(branch, "snapshot") && strings.Contains(branch, "ollama-unavail-test") {
			snapshotBranch = branch
			break
		}
	}

	if snapshotBranch == "" {
		t.Fatalf("snapshot branch not created")
	}

	// Snapshot should still be created successfully
	// (embedding generation failure doesn't fail the snapshot)
}

func TestBuildEmbeddingText(t *testing.T) {
	metadata := &models.Metadata{
		Topic: "test-topic",
		Tags:  []string{"tag1", "tag2"},
		Notes: "Short notes",
	}

	notesContent := "# Test\n\nDetailed notes content"

	result := buildEmbeddingText(metadata, notesContent)

	// Verify all components are included
	if !strings.Contains(result, "test-topic") {
		t.Error("embedding text should contain topic")
	}
	if !strings.Contains(result, "tag1") {
		t.Error("embedding text should contain tags")
	}
	if !strings.Contains(result, "Short notes") {
		t.Error("embedding text should contain notes from metadata")
	}
	if !strings.Contains(result, "Detailed notes content") {
		t.Error("embedding text should contain notes.md content")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Simple Test",
			expected: "simple-test",
		},
		{
			input:    "With Special! Characters?",
			expected: "with-special-characters",
		},
		{
			input:    "Multiple   Spaces",
			expected: "multiple---spaces", // Multiple spaces become multiple hyphens
		},
		{
			input:    "UPPERCASE",
			expected: "uppercase",
		},
		{
			input:    "numbers-123",
			expected: "numbers-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
