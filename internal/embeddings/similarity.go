package embeddings

import (
	"fmt"
	"math"
)

// CosineSimilarity calculates the cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 means identical direction
func CosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have same length: %d vs %d", len(a), len(b))
	}

	if len(a) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}

	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("vector norm cannot be zero")
	}

	similarity := dotProduct / (normA * normB)

	// Clamp to [-1, 1] to handle floating point errors
	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}

	return similarity, nil
}

// Normalize normalizes a vector to unit length
func Normalize(v []float64) ([]float64, error) {
	if len(v) == 0 {
		return nil, fmt.Errorf("vector cannot be empty")
	}

	norm := 0.0
	for _, val := range v {
		norm += val * val
	}
	norm = math.Sqrt(norm)

	if norm == 0 {
		return nil, fmt.Errorf("cannot normalize zero vector")
	}

	result := make([]float64, len(v))
	for i, val := range v {
		result[i] = val / norm
	}

	return result, nil
}

// DotProduct calculates the dot product of two vectors
func DotProduct(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have same length: %d vs %d", len(a), len(b))
	}

	if len(a) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}

	result := 0.0
	for i := range a {
		result += a[i] * b[i]
	}

	return result, nil
}

// Magnitude calculates the magnitude (Euclidean norm) of a vector
func Magnitude(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}

	sum := 0.0
	for _, val := range v {
		sum += val * val
	}

	return math.Sqrt(sum)
}
