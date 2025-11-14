# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**git-context** is a CLI tool for creating temporal, immutable research snapshots in Git repositories. Each snapshot captures codebase state + research artifacts (notes, POCs, analyses) as an immutable Git branch with structured metadata. Optimized for both human developers and AI agents.

## Development Commands

```bash
# Build
go build -o context

# Install locally
go install

# Run full test suite
go test ./...

# Run specific package tests
go test ./cmd -v
go test ./internal/embeddings -v

# Run single test
go test -v ./cmd -run TestSaveFullMode

# Run with coverage
go test ./... -cover

# Run with timeout
go test -v ./cmd -timeout 30s
```

## Architecture

### Core Concept: Worktree-Based Snapshots

**Critical design decision**: Snapshots are created in temporary worktrees (`/tmp/context-snapshot-<unix-timestamp>`), never in the user's working directory. This is fundamental to the safety model.

**Flow**:
1. User runs `context save "topic"`
2. Create new branch: `snapshot/YYYY-MM-DDTHHMM/topic-slug`
3. Create temporary worktree for that branch
4. Populate research artifacts in `research/YYYY-MM-DDTHHMM/topic-slug/`
5. Apply snapshot mode (full/research-only/diff/poc)
6. Commit in worktree using `git.CommitInDirNoVerify()` (bypasses pre-commit hook)
7. Remove worktree
8. Snapshot branch now exists, user's working directory untouched

**Why worktrees?**
- Zero risk to user's current work
- No checkout dance or stashing
- Clean separation between snapshot creation and user workspace
- Consistent with `context open` which also uses worktrees

### Immutability Protection

Three layers:
1. **Pre-commit hook** (`hooks/pre-commit`) - Blocks any commits to `snapshot/*` branches
2. **CLI validation** - `context save` refuses to overwrite existing snapshot branches
3. **Metadata integrity** - Tree hash stored in `meta.json` for verification

**Important**: The save command uses `git.CommitInDirNoVerify()` (with `--no-verify` flag) to bypass the pre-commit hook when creating the initial snapshot. This is intentional and safe because we control the operation.

### Git Operations in Custom Directories

All git operations in `internal/git/git.go` support working in custom directories:
- `AddFilesInDir(dir, paths...)`
- `CommitInDir(dir, message)`
- `CommitInDirNoVerify(dir, message)` - Uses `--no-verify` to bypass hooks
- `RemoveAllFilesFromIndexInDir(dir)`

This is essential for the worktree-based architecture.

### Snapshot Modes

Implemented in `cmd/save.go` switch statement:
- **full**: Complete codebase + research (just add research dir)
- **research-only**: Only research + commit ref (remove all code from index, add research)
- **diff**: Patch file + research (create `changes.patch`, remove code, add research)
- **poc**: Selected files + research (remove all, add research, add specified files)

### Metadata Model

Defined in `internal/models/metadata.go`:
```go
type Metadata struct {
    CreatedAt     time.Time
    Topic         string
    Root          string       // Branch name
    Mode          SnapshotMode
    RelatedBranch string       // Source branch
    MainCommit    string       // Source commit
    Tags          []string
    Embedding     string       // Filename of embedding.bin
    Notes         string
    TreeHash      string       // For integrity verification
    RelatedTo     []string     // Explicit relationships to other snapshots
}
```

Stored as `research/YYYY-MM-DDTHHMM/topic-slug/meta.json`

### Embeddings & Semantic Search

**Ollama Integration** (`internal/ollama/client.go`):
- Wraps Ollama API at `http://localhost:11434`
- Default model: `nomic-embed-text`
- Graceful degradation when Ollama unavailable

**Embedding Generation** (`cmd/save.go`):
1. Combine topic + tags + notes + notes.md content
2. Truncate to ~30K characters (model limit)
3. Call Ollama API
4. Store as binary float64 array in `embedding.bin` (LittleEndian)
5. Reference in metadata

**Hybrid Search** (`cmd/search.go`):
- Keyword search (30%): Matches in topic, tags, notes, related branch
- Semantic search (70%): Cosine similarity of embeddings
- Falls back to keyword-only when embeddings unavailable
- Configurable weights in config.toml

**Boolean Operators**:
- `+term` - Required (must be present)
- `-term` - Excluded (must NOT be present)
- `"exact phrase"` - Exact phrase match
- Implemented via `parseSearchQuery()` returning `searchQuery` struct
- `calculateRelevance()` returns `(score int, shouldExclude bool)`

### Agent-Friendly Design

**All commands support machine-readable output**:
- `--json` flag - Standard JSON
- `--toon` flag - LLM-optimized format via `github.com/alpkeskin/gotoon`

**Output patterns**:
```go
// Check flags before processing
if cmdJSON {
    output, _ := json.MarshalIndent(data, "", "  ")
    fmt.Println(string(output))
    return nil
}

if cmdToon {
    output, _ := gotoon.Encode(data)
    fmt.Println(output)
    return nil
}

// Human-readable output...
```

Conditional message display:
```go
if !searchJSON && !searchToon {
    fmt.Println("Using hybrid search (keyword + semantic)")
}
```

### Relationships & Analytics

**Explicit relationships** (`--related-to` flag):
- Stored in `metadata.RelatedTo` array
- References to other snapshot branches

**Relationship discovery** (`cmd/related.go`):
- Explicit links: 100 points
- Shared tags: 10 points each
- Same topic: 20 points

**Multi-tag filtering** (`cmd/list.go`):
- Multiple `--tag` flags use AND logic
- Must have ALL specified tags

**Tag management** (`cmd/tags.go`):
- List all tags with counts
- Rename tags across all snapshots atomically using worktrees

**Statistics** (`cmd/stats.go`):
- Aggregate counts by mode, tag, date
- Embedding coverage
- Top tags
- Daily activity bars

### Testing Patterns

**Test Isolation** (`internal/testutil/testutil.go`):
```go
repo := testutil.NewTempGitRepo(t)
defer repo.Cleanup()
// ... test in isolated repo
```

Each test gets a temporary Git repository with initial commit, ensuring complete isolation.

**Mock Ollama Server** (`internal/ollama/client_test.go`):
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Return mock embedding
}))
defer server.Close()
```

Tests that require Ollama use mock HTTP servers.

## Key Implementation Patterns

### Branch Naming Convention
```
snapshot/YYYY-MM-DDTHHMM/topic-slug
```
Generated by `models.BranchName(timestamp, topic)` - always use this, never construct manually.

### Research Directory Structure
```
research/YYYY-MM-DDTHHMM/topic-slug/
├── notes.md
├── meta.json
├── embedding.bin (optional)
└── ...other artifacts
```
Generated by `models.ResearchPath(timestamp, topic)` and `models.MetadataPath(timestamp, topic)`.

### Configuration
Uses Viper for config management. Defaults in `cmd/root.go`:
```go
viper.SetDefault("retention.days", 90)
viper.SetDefault("embeddings.enabled", true)
viper.SetDefault("search.keyword_weight", 0.3)
viper.SetDefault("search.semantic_weight", 0.7)
```

Config file: `~/.config/git-context/config.toml`

### Shared Helper Functions

**`parseSnapshotBranch(branch string)`** - Used by list, meta, related, diff, stats:
```go
// Returns snapshotInfo with Branch, Timestamp, Topic
info, err := parseSnapshotBranch("snapshot/2025-11-14T0930/topic")
```

**`snapshotInfo.LoadMetadata()`** - Lazy loads metadata:
```go
info.LoadMetadata()  // Populates info.Metadata and info.HasEmbedding
```

**`gitShow(branch, path)`** - Read file from specific branch:
```go
content, err := gitShow("snapshot/2025-11-14T0930/topic", "research/.../meta.json")
```

## Important Constraints

1. **Never checkout snapshot branches** - Use worktrees or `git show`
2. **Always use `CommitInDirNoVerify()` for snapshot creation** - Must bypass hook
3. **Worktree cleanup is critical** - Always defer `git.RemoveWorktree()`
4. **Snapshot branches are immutable** - Refuse overwrites, respect hook
5. **Graceful degradation** - All embedding features must work without Ollama
6. **Binary format matters** - Embeddings use LittleEndian float64 encoding

## Common Extension Points

**New commands** - Add to `cmd/` following pattern:
1. Create `cmd/newcmd.go` with `cobra.Command`
2. Add to `init()` function: `rootCmd.AddCommand(newCmd)`
3. Support `--json` and `--toon` flags if outputting data
4. Add corresponding `cmd/newcmd_test.go`

**New snapshot modes** - Modify `cmd/save.go`:
1. Add to `SnapshotMode` constants in `internal/models/metadata.go`
2. Add case to switch statement in `runSave()`
3. Update `isValidMode()` validation

**New report templates** - Add to `cmd/report.go`:
1. Add case to switch in `runReport()`
2. Implement function that calls existing commands with specific flags
3. Follow pattern: set flags, call command, restore flags

## Gotchas

- **Don't use `git add .` in worktrees** - Use `git.AddFilesInDir()` with specific paths
- **Test for Ollama availability** - Use `ollama.IsAvailable(url)` before embedding operations
- **Handle nil metadata** - Many snapshots may not have metadata loaded yet
- **Pre-commit hook blocks snapshot commits** - This is intentional; use `--no-verify` only in controlled operations
- **Tag rename is not instant** - Iterates through all snapshot branches using worktrees
