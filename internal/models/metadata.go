package models

import "time"

// SnapshotMode defines the type of snapshot
type SnapshotMode string

const (
	ModeFull         SnapshotMode = "full"
	ModeResearchOnly SnapshotMode = "research-only"
	ModeDiff         SnapshotMode = "diff"
	ModePOC          SnapshotMode = "poc"
)

// Metadata represents the meta.json structure for a snapshot
type Metadata struct {
	CreatedAt     time.Time    `json:"created_at"`
	Topic         string       `json:"topic"`
	Root          string       `json:"root"`
	Mode          SnapshotMode `json:"mode"`
	RelatedBranch string       `json:"related_branch,omitempty"`
	MainCommit    string       `json:"main_commit"`
	Tags          []string     `json:"tags,omitempty"`
	Embedding     string       `json:"embedding,omitempty"`
	Notes         string       `json:"notes,omitempty"`
	TreeHash      string       `json:"tree_hash,omitempty"` // For immutability verification
}
