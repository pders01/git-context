package embeddings

import (
	"math"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		wantErr  bool
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 2.0, 3.0},
			b:        []float64{1.0, 2.0, 3.0},
			expected: 1.0,
			wantErr:  false,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{0.0, 1.0},
			expected: 0.0,
			wantErr:  false,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{-1.0, 0.0},
			expected: -1.0,
			wantErr:  false,
		},
		{
			name:     "similar vectors",
			a:        []float64{1.0, 2.0, 3.0},
			b:        []float64{2.0, 4.0, 6.0},
			expected: 1.0,
			wantErr:  false,
		},
		{
			name:     "different length vectors",
			a:        []float64{1.0, 2.0},
			b:        []float64{1.0, 2.0, 3.0},
			expected: 0.0,
			wantErr:  true,
		},
		{
			name:     "zero vectors",
			a:        []float64{0.0, 0.0},
			b:        []float64{1.0, 2.0},
			expected: 0.0,
			wantErr:  true,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CosineSimilarity(tt.a, tt.b)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		wantErr  bool
	}{
		{
			name:     "simple vectors",
			a:        []float64{1.0, 2.0, 3.0},
			b:        []float64{4.0, 5.0, 6.0},
			expected: 32.0, // 1*4 + 2*5 + 3*6 = 4 + 10 + 18 = 32
			wantErr:  false,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{0.0, 1.0},
			expected: 0.0,
			wantErr:  false,
		},
		{
			name:     "negative values",
			a:        []float64{-1.0, 2.0},
			b:        []float64{3.0, -4.0},
			expected: -11.0, // -1*3 + 2*-4 = -3 + -8 = -11
			wantErr:  false,
		},
		{
			name:     "different length vectors",
			a:        []float64{1.0, 2.0},
			b:        []float64{1.0, 2.0, 3.0},
			expected: 0.0,
			wantErr:  true,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DotProduct(tt.a, tt.b)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestMagnitude(t *testing.T) {
	tests := []struct {
		name     string
		v        []float64
		expected float64
	}{
		{
			name:     "unit vector",
			v:        []float64{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "3-4-5 triangle",
			v:        []float64{3.0, 4.0},
			expected: 5.0,
		},
		{
			name:     "all ones",
			v:        []float64{1.0, 1.0, 1.0},
			expected: math.Sqrt(3.0),
		},
		{
			name:     "zero vector",
			v:        []float64{0.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "negative values",
			v:        []float64{-3.0, 4.0},
			expected: 5.0,
		},
		{
			name:     "empty vector",
			v:        []float64{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Magnitude(tt.v)

			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		v       []float64
		wantErr bool
	}{
		{
			name:    "simple vector",
			v:       []float64{3.0, 4.0},
			wantErr: false,
		},
		{
			name:    "unit vector",
			v:       []float64{1.0, 0.0, 0.0},
			wantErr: false,
		},
		{
			name:    "all ones",
			v:       []float64{1.0, 1.0, 1.0},
			wantErr: false,
		},
		{
			name:    "zero vector",
			v:       []float64{0.0, 0.0},
			wantErr: true,
		},
		{
			name:    "empty vector",
			v:       []float64{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Normalize(tt.v)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check that magnitude is 1.0
			mag := Magnitude(result)
			if math.Abs(mag-1.0) > 1e-10 {
				t.Errorf("normalized vector should have magnitude 1.0, got %f", mag)
			}

			// Check that the vector is in the same direction
			// (cosine similarity with original should be 1.0 or close)
			if len(tt.v) > 0 && Magnitude(tt.v) > 0 {
				sim, err := CosineSimilarity(tt.v, result)
				if err != nil {
					t.Errorf("failed to calculate similarity: %v", err)
					return
				}
				if math.Abs(sim-1.0) > 1e-10 {
					t.Errorf("normalized vector should point in same direction, similarity: %f", sim)
				}
			}
		})
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	// Benchmark with typical embedding size (768 dimensions)
	a := make([]float64, 768)
	vec := make([]float64, 768)
	for i := range a {
		a[i] = float64(i) / 768.0
		vec[i] = float64(i+1) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CosineSimilarity(a, vec)
	}
}

func BenchmarkNormalize(b *testing.B) {
	v := make([]float64, 768)
	for i := range v {
		v[i] = float64(i) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Normalize(v)
	}
}
