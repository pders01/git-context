package models

import (
	"fmt"
	"time"
)

// Snapshot represents a temporal research snapshot
type Snapshot struct {
	Timestamp time.Time
	Topic     string
	Branch    string
	Metadata  *Metadata
}

// BranchName generates the branch name from timestamp and topic
// Format: snapshot/YYYY-MM-DDTHHMM/topic-slug
func BranchName(timestamp time.Time, topic string) string {
	return fmt.Sprintf("snapshot/%s/%s",
		timestamp.Format("2006-01-02T1504"),
		topic,
	)
}

// ResearchPath generates the research directory path
// Format: research/YYYY-MM-DDTHHMM/topic-slug
func ResearchPath(timestamp time.Time, topic string) string {
	return fmt.Sprintf("research/%s/%s",
		timestamp.Format("2006-01-02T1504"),
		topic,
	)
}

// MetadataPath returns the path to meta.json for a snapshot
func MetadataPath(timestamp time.Time, topic string) string {
	return fmt.Sprintf("%s/meta.json", ResearchPath(timestamp, topic))
}
