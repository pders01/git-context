package embeddings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadEmbedding(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "embedding-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		embedding []float64
		wantErr   bool
	}{
		{
			name:      "simple embedding",
			embedding: []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			wantErr:   false,
		},
		{
			name:      "large embedding (768 dimensions)",
			embedding: generateTestEmbedding(768),
			wantErr:   false,
		},
		{
			name:      "small embedding",
			embedding: []float64{0.5},
			wantErr:   false,
		},
		{
			name:      "negative values",
			embedding: []float64{-1.0, -2.0, -3.0},
			wantErr:   false,
		},
		{
			name:      "mixed values",
			embedding: []float64{-1.5, 0.0, 1.5, 2.5, -3.5},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".bin")

			// Write embedding
			err := WriteEmbedding(path, tt.embedding)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to write embedding: %v", err)
			}

			// Read embedding back
			result, err := ReadEmbedding(path)
			if err != nil {
				t.Fatalf("failed to read embedding: %v", err)
			}

			// Compare
			if len(result) != len(tt.embedding) {
				t.Errorf("expected length %d, got %d", len(tt.embedding), len(result))
				return
			}

			for i := range tt.embedding {
				if result[i] != tt.embedding[i] {
					t.Errorf("mismatch at index %d: expected %f, got %f", i, tt.embedding[i], result[i])
				}
			}
		})
	}
}

func TestReadNonexistentFile(t *testing.T) {
	_, err := ReadEmbedding("/nonexistent/path/embedding.bin")
	if err == nil {
		t.Error("expected error when reading nonexistent file")
	}
}

func TestWriteToInvalidPath(t *testing.T) {
	// Try to write to a directory that doesn't exist
	err := WriteEmbedding("/nonexistent/directory/embedding.bin", []float64{1.0, 2.0})
	if err == nil {
		t.Error("expected error when writing to invalid path")
	}
}

func TestValidateEmbedding(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float64
		wantErr   bool
	}{
		{
			name:      "valid embedding",
			embedding: []float64{1.0, 2.0, 3.0},
			wantErr:   false,
		},
		{
			name:      "valid large embedding",
			embedding: generateTestEmbedding(768),
			wantErr:   false,
		},
		{
			name:      "empty embedding",
			embedding: []float64{},
			wantErr:   true,
		},
		{
			name:      "nil embedding",
			embedding: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmbedding(tt.embedding)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEmbeddingFileSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "embedding-size-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test that file size is correct (8 bytes per float64)
	embedding := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	path := filepath.Join(tmpDir, "test.bin")

	if err := WriteEmbedding(path, embedding); err != nil {
		t.Fatalf("failed to write embedding: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	expectedSize := int64(len(embedding) * 8)
	if info.Size() != expectedSize {
		t.Errorf("expected file size %d bytes, got %d bytes", expectedSize, info.Size())
	}
}

func TestReadCorruptedFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "embedding-corrupt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write a file with incomplete float64 (not multiple of 8 bytes)
	path := filepath.Join(tmpDir, "corrupt.bin")
	corruptData := []byte{0x01, 0x02, 0x03, 0x04, 0x05} // Only 5 bytes

	if err := os.WriteFile(path, corruptData, 0644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	_, err = ReadEmbedding(path)
	if err == nil {
		t.Error("expected error when reading corrupted file")
	}
}

// generateTestEmbedding creates a test embedding of the specified size
func generateTestEmbedding(size int) []float64 {
	vec := make([]float64, size)
	for i := range vec {
		vec[i] = float64(i) / float64(size)
	}
	return vec
}

func BenchmarkWriteEmbedding(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "embedding-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	embedding := generateTestEmbedding(768)
	path := filepath.Join(tmpDir, "bench.bin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WriteEmbedding(path, embedding)
	}
}

func BenchmarkReadEmbedding(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "embedding-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	embedding := generateTestEmbedding(768)
	path := filepath.Join(tmpDir, "bench.bin")

	if err := WriteEmbedding(path, embedding); err != nil {
		b.Fatalf("failed to write embedding: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ReadEmbedding(path)
	}
}
