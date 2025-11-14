# git-context

**Temporal, immutable research snapshots for Git repositories.**

git-context creates Git-native snapshots that capture your codebase state alongside research artifacts (notes, POCs, analyses). Each snapshot is immutable, searchable, and optimized for both human developers and AI agents.

## Why?

Traditional documentation drifts. Code and notes become desynchronized. Important context gets lost.

git-context solves this by:
- **Perfect synchronization** - Code and research captured atomically
- **Immutability** - Snapshots never change, preserving context forever
- **Discoverability** - Tag-based organization, semantic search, relationship tracking
- **Agent-friendly** - JSON/Toon output everywhere for programmatic access

## Quick Start

```bash
# Install (requires Go)
go install github.com/pders01/git-context@latest

# Initialize in your repo
context init

# Create a snapshot
context save "security-audit" --tag security --tag important \
  --notes "Found potential XSS vulnerability in user input handling"

# List snapshots
context list

# Search
context search "vulnerability"

# Daily standup report
context report daily

# View relationships
context related 2025-11-14T0930 security-audit
```

## Core Concepts

### Snapshots

Snapshots are immutable Git branches following the pattern:
```
snapshot/YYYY-MM-DDTHHMM/topic-slug
```

Example: `snapshot/2025-11-14T0930/security-audit`

Each snapshot contains:
- Research artifacts in `research/YYYY-MM-DDTHHMM/topic-slug/`
- Structured metadata (`meta.json`)
- Optional codebase (depending on mode)
- Optional vector embeddings for semantic search

### Snapshot Modes

| Mode | Contents | Use Case |
|------|----------|----------|
| `full` | Complete codebase + research | Default, full context preservation |
| `research-only` | Only research artifacts + commit ref | Lightweight, minimal disk usage |
| `diff` | Patch file + research + commit ref | Code changes documentation |
| `poc` | Selected files + research | Proof-of-concept tracking |

### Tags

Organize snapshots with tags for easy filtering and grouping:
```bash
context save "analysis" --tag security --tag important --tag phase2
context list --tag security --tag important  # AND logic
context tags                                   # List all tags with counts
```

### Relationships

Link related snapshots to track research progressions:
```bash
context save "follow-up-analysis" \
  --related-to snapshot/2025-11-14T0930/initial-investigation

context related 2025-11-14T0930 initial-investigation
```

## Commands

### context save

Create a new snapshot:

```bash
# Basic snapshot
context save "migration-notes"

# Research-only mode (no code)
context save "analysis" --mode research-only

# With tags and notes
context save "security-audit" \
  --tag security --tag important \
  --notes "Found CVE-2024-xxxx in dependency"

# Link to related snapshots
context save "follow-up" \
  --related-to snapshot/2025-11-14T0930/initial-recon

# POC mode (specific files only)
context save "prototype" --mode poc \
  --include main.go --include proto.go

# Skip embedding generation (faster)
context save "quick-snapshot" --no-embed

# Batch create multiple snapshots
context save --batch snapshots.toml
```

### context list

List and filter snapshots:

```bash
# List all
context list

# Filter by topic
context list --topic security

# Filter by tag (AND logic)
context list --tag security --tag important

# Filter by date
context list --today
context list --since 2025-10-01

# Group output
context list --group-by tag
context list --group-by date
context list --group-by mode

# Machine-readable output
context list --json
context list --toon
```

### context search

Hybrid keyword + semantic search:

```bash
# Basic search
context search "vulnerability"

# With topic filter
context search --topic security "authentication"

# Boolean operators
context search "+security +important"           # Both required
context search "bug -deprecated"                # Exclude deprecated
context search "\"exact phrase match\""         # Exact phrase

# Machine-readable
context search "query" --json
context search "query" --toon
```

Search automatically uses semantic search (via Ollama embeddings) when available, falling back to keyword search gracefully.

### context tags

Tag management and discovery:

```bash
# List all tags with counts
context tags

# Rename tag across all snapshots
context tags old-name --rename new-name

# JSON output
context tags --json
```

### context related

Find related snapshots:

```bash
# Find snapshots related to a specific one
context related 2025-11-14T0930 security-audit

# JSON output
context related 2025-11-14T0930 topic --json
```

Scoring algorithm:
- Explicit relationships: 100 points
- Shared tags: 10 points each
- Same topic: 20 points

### context diff

Compare two snapshots:

```bash
# Compare snapshots
context diff 2025-11-14T0930 initial-recon \
             2025-11-14T1015 vulnerability-analysis

# JSON output
context diff <timestamp1> <topic1> <timestamp2> <topic2> --json
```

Shows:
- Time difference
- Tag changes (added/removed/shared)
- Mode changes
- Notes changes
- Commit changes

### context stats

Analytics dashboard:

```bash
# View statistics
context stats

# JSON for programmatic access
context stats --json
```

Shows:
- Total snapshots, date range
- Breakdown by mode
- Embedding coverage
- Top tags
- Recent activity (last 7 days)

### context report

Pre-formatted reports:

```bash
# Daily standup report
context report daily
```

Combines stats + today's snapshots grouped by tag.

### context meta

Show snapshot metadata:

```bash
context meta 2025-11-14T0930 security-audit

# JSON output
context meta 2025-11-14T0930 security-audit --json
```

### context open

Open snapshot in worktree:

```bash
context open 2025-11-14T0930 security-audit

# Custom path
context open 2025-11-14T0930 security-audit --path /tmp/my-snapshot
```

### context prune

Apply retention policy:

```bash
# Dry run (preview)
context prune

# Actually delete
context prune --force
```

Configurable via `~/.config/context/config.toml`:
```toml
[retention]
days = 90
preserve_tags = ["important", "security"]
```

### context archive

Bundle snapshots for backup:

```bash
# Archive by year
context archive 2024

# Archive by month
context archive 2024-11

# Archive all
context archive all

# Filter by topic
context archive 2024 --topic security --output backups.tar.gz
```

### context init

Initialize repository (one-time setup):

```bash
context init
```

Installs pre-commit hook to prevent accidental snapshot modifications.

## Agent-Friendly Features

Every command supports machine-readable output:

```bash
# JSON output
context list --json | jq '.[] | select(.metadata.tags | contains(["important"]))'
context search "query" --json | jq '.[].score'
context stats --json | jq '.top_tags[:5]'

# Toon output (LLM-optimized format)
context list --toon
context search "query" --toon
context related <timestamp> <topic> --toon
```

### Batch Operations

Create multiple snapshots from TOML config:

```toml
# snapshots.toml
[[snapshot]]
topic = "initial-recon"
mode = "research-only"
tags = ["security", "investigation"]
notes = "Initial security audit findings"

[[snapshot]]
topic = "vulnerability-analysis"
mode = "research-only"
tags = ["security", "important"]
notes = "Confirmed XSS vulnerability"
related_to = ["snapshot/2025-11-14T0930/initial-recon"]
```

```bash
context save --batch snapshots.toml
```

## Semantic Search with Ollama

git-context integrates with [Ollama](https://ollama.ai) for local, private semantic search.

### Setup

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull embedding model
ollama pull nomic-embed-text
```

### Usage

Snapshots automatically generate embeddings when created. Search uses hybrid approach:
- 30% keyword matching
- 70% semantic similarity

```bash
# This uses semantic search when Ollama is running
context search "authentication vulnerabilities"

# Configure weights
~/.config/context/config.toml:
[search]
keyword_weight = 0.3
semantic_weight = 0.7
```

Gracefully degrades to keyword-only when Ollama unavailable.

## Configuration

Create `~/.config/context/config.toml`:

```toml
[retention]
days = 90
preserve_tags = ["important", "security", "milestone"]

[snapshot]
default_mode = "research-only"
research_dir = "research"

[embeddings]
enabled = true
model = "nomic-embed-text"
ollama_url = "http://localhost:11434"

[search]
keyword_weight = 0.3
semantic_weight = 0.7
```

## Architecture Highlights

### Immutability

Three layers of protection:
1. **Pre-commit hook** - Prevents commits to `snapshot/*` branches
2. **CLI validation** - Refuses to overwrite existing snapshots
3. **Metadata integrity** - Tree hash verification

### Worktree-Based Snapshots

Snapshots created in temporary worktrees, never touching your working directory:
- Zero risk to current work
- Clean separation of concerns
- Automatic cleanup

### Git-Native

Everything is standard Git:
- Branches for snapshots
- No custom database
- Works with existing Git tools
- Push/pull friendly

## Real-World Workflows

### Security Research

```bash
# Initial investigation
context save "initial-recon" --tag security --tag recon \
  --notes "Reviewing authentication code for vulnerabilities"

# Deep analysis
context save "vulnerability-analysis" --tag security --tag important \
  --related-to snapshot/2025-11-14T0930/initial-recon \
  --notes "Confirmed XSS in user input handling"

# Track progression
context diff 2025-11-14T0930 initial-recon 2025-11-14T1015 vulnerability-analysis
context related 2025-11-14T0930 initial-recon
```

### Daily Standup

```bash
# What did I work on today?
context report daily

# Or manual approach
context list --today --group-by tag
context stats
```

### Tag Analytics

```bash
# What are my common tags?
context tags

# Find all milestone work
context list --tag milestone --group-by date

# Generate metrics
context stats --json | jq '.top_tags[:10]'
```

## Testing

```bash
# Run full test suite
go test ./...

# Specific package
go test ./cmd -v

# With coverage
go test ./... -cover
```

## Development

```bash
# Build
go build -o context

# Install locally
go install

# Run tests
go test ./...
```

## Project Structure

```
git-context/
├── cmd/                    # CLI commands
│   ├── archive.go         # Backup/export
│   ├── diff.go            # Snapshot comparison
│   ├── init.go            # Repository initialization
│   ├── list.go            # Listing and filtering
│   ├── meta.go            # Metadata display
│   ├── open.go            # Worktree management
│   ├── prune.go           # Retention policy
│   ├── related.go         # Relationship discovery
│   ├── report.go          # Report generation
│   ├── root.go            # Root command & config
│   ├── save.go            # Snapshot creation
│   ├── search.go          # Hybrid search
│   ├── stats.go           # Analytics
│   └── tags.go            # Tag management
├── internal/
│   ├── config/            # Configuration
│   ├── embeddings/        # Vector operations
│   ├── git/               # Git operations
│   ├── models/            # Data structures
│   ├── ollama/            # Ollama integration
│   └── testutil/          # Test utilities
├── hooks/
│   └── pre-commit         # Immutability protection
└── main.go
```

## Contributing

This project is a personal tool but contributions are welcome. Areas of interest:
- Additional report templates
- Enhanced semantic search
- Web UI for browsing snapshots
- Integration with other tools

## License

MIT

## Credits

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration
- [Ollama](https://ollama.ai) - Local embeddings
- [gotoon](https://github.com/alpkeskin/gotoon) - Toon format encoding
