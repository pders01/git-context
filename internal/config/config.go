package config

import (
	"github.com/paulderscheid/git-context/internal/models"
	"github.com/spf13/viper"
)

// GetRetentionDays returns the retention period in days
func GetRetentionDays() int {
	return viper.GetInt("retention.days")
}

// GetPreserveTags returns tags that should be preserved indefinitely
func GetPreserveTags() []string {
	return viper.GetStringSlice("retention.preserve_tags")
}

// GetDefaultMode returns the default snapshot mode
func GetDefaultMode() models.SnapshotMode {
	mode := viper.GetString("snapshot.default_mode")
	return models.SnapshotMode(mode)
}

// GetResearchDir returns the research directory name
func GetResearchDir() string {
	return viper.GetString("snapshot.research_dir")
}

// ShouldPreserve checks if a snapshot with given tags should be preserved
func ShouldPreserve(tags []string) bool {
	preserveTags := GetPreserveTags()
	for _, tag := range tags {
		for _, preserveTag := range preserveTags {
			if tag == preserveTag {
				return true
			}
		}
	}
	return false
}

// GetEmbeddingsEnabled returns whether embeddings are enabled
func GetEmbeddingsEnabled() bool {
	return viper.GetBool("embeddings.enabled")
}

// GetEmbeddingModel returns the embedding model to use
func GetEmbeddingModel() string {
	model := viper.GetString("embeddings.model")
	if model == "" {
		return "nomic-embed-text"
	}
	return model
}

// GetOllamaURL returns the Ollama API endpoint
func GetOllamaURL() string {
	url := viper.GetString("embeddings.ollama_url")
	if url == "" {
		return "http://localhost:11434"
	}
	return url
}

// GetKeywordWeight returns the weight for keyword scoring in hybrid search
func GetKeywordWeight() float64 {
	weight := viper.GetFloat64("search.keyword_weight")
	if weight == 0 {
		return 0.3 // default
	}
	return weight
}

// GetSemanticWeight returns the weight for semantic scoring in hybrid search
func GetSemanticWeight() float64 {
	weight := viper.GetFloat64("search.semantic_weight")
	if weight == 0 {
		return 0.7 // default
	}
	return weight
}
