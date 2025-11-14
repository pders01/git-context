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
