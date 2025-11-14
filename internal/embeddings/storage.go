package embeddings

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// WriteEmbedding writes an embedding vector to a binary file
// Format: LittleEndian float64 array
func WriteEmbedding(path string, vec []float64) error {
	if len(vec) == 0 {
		return fmt.Errorf("embedding vector cannot be empty")
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create embedding file: %w", err)
	}
	defer file.Close()

	// Write each float64 value
	for _, val := range vec {
		if err := binary.Write(file, binary.LittleEndian, val); err != nil {
			return fmt.Errorf("failed to write embedding value: %w", err)
		}
	}

	return nil
}

// ReadEmbedding reads an embedding vector from a binary file
func ReadEmbedding(path string) ([]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedding file: %w", err)
	}
	defer file.Close()

	// Get file size to calculate vector length
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat embedding file: %w", err)
	}

	size := stat.Size()
	if size == 0 {
		return nil, fmt.Errorf("embedding file is empty")
	}

	// Each float64 is 8 bytes
	if size%8 != 0 {
		return nil, fmt.Errorf("invalid embedding file size: %d (not a multiple of 8)", size)
	}

	vectorLen := size / 8
	vec := make([]float64, vectorLen)

	// Read each float64 value
	for i := range vec {
		if err := binary.Read(file, binary.LittleEndian, &vec[i]); err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected EOF at element %d", i)
			}
			return nil, fmt.Errorf("failed to read embedding value at %d: %w", i, err)
		}
	}

	return vec, nil
}

// EmbeddingSize returns the size in bytes of an embedding file
func EmbeddingSize(dimensions int) int64 {
	// Each float64 is 8 bytes
	return int64(dimensions * 8)
}

// ValidateEmbedding checks if an embedding vector is valid
func ValidateEmbedding(vec []float64) error {
	if len(vec) == 0 {
		return fmt.Errorf("embedding vector is empty")
	}

	// Check for NaN or Inf values
	for i, val := range vec {
		if val != val { // NaN check
			return fmt.Errorf("embedding contains NaN at index %d", i)
		}
		if val > 1e308 || val < -1e308 { // Rough Inf check
			return fmt.Errorf("embedding contains invalid value at index %d: %v", i, val)
		}
	}

	return nil
}
