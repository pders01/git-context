package ollama

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ollama/ollama/api"
)

const (
	// DefaultModel is the recommended embedding model
	DefaultModel = "nomic-embed-text"
	// DefaultURL is the default Ollama API endpoint
	DefaultURL = "http://localhost:11434"
)

// Client wraps the Ollama API client
type Client struct {
	client *api.Client
	model  string
}

// NewClient creates a new Ollama client
func NewClient(url, model string) (*Client, error) {
	if url == "" {
		url = DefaultURL
	}
	if model == "" {
		model = DefaultModel
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama client: %w", err)
	}

	return &Client{
		client: client,
		model:  model,
	}, nil
}

// IsAvailable checks if Ollama is running and accessible
func IsAvailable(url string) bool {
	if url == "" {
		url = DefaultURL
	}

	// Try to connect with a short timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GenerateEmbedding generates an embedding vector for the given text
func (c *Client) GenerateEmbedding(text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	ctx := context.Background()

	req := &api.EmbedRequest{
		Model: c.model,
		Input: text,
	}

	resp, err := c.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert from []float32 to []float64
	embedding32 := resp.Embeddings[0]
	embedding64 := make([]float64, len(embedding32))
	for i, v := range embedding32 {
		embedding64[i] = float64(v)
	}

	return embedding64, nil
}

// CheckModel checks if the specified model is available
func (c *Client) CheckModel() error {
	ctx := context.Background()

	// List available models
	listResp, err := c.client.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	// Check if our model is in the list
	for _, model := range listResp.Models {
		if model.Name == c.model {
			return nil
		}
	}

	return fmt.Errorf("model '%s' not found - run: ollama pull %s", c.model, c.model)
}

// GetModel returns the model being used
func (c *Client) GetModel() string {
	return c.model
}
