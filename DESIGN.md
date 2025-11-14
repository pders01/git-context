# **Temporal Snapshot Workflow — Final Design Document**

*Version 1.0 — Implementation Ready*

---

# **1. Purpose**

This document defines a Git-native workflow and associated CLI tool for creating **temporal, immutable research snapshots**. Each snapshot captures:

* the exact codebase state
* research artifacts (notes, POCs, analyses)
* optional metadata
* optional vector embeddings for agentic search

This system eliminates documentation drift, preserves rationale, and supports both human developers and agentic tools.

---

# **2. Goals**

### **Primary Goals**

* Perfect synchronization between research and code
* Zero manual cognitive load
* Immutable snapshots that preserve context
* First-class support for automated agentic tooling
* CLI tooling with a clean UX

### **Secondary Goals**

* Structured search
* Timeline navigation
* Automatic naming and grouping
* Optional lightweight archival mode
* Optional semantic search through embeddings

---

# **3. Snapshot Model**

### **3.1 Branch Naming Convention**

Use a timestamp + topic grouping:

```
snapshot/YYYY-MM-DDTHHMM/topic-slug
```

Examples:

```
snapshot/2025-11-14T0930/security-audit
snapshot/2025-11-14T1502/pnpm-migration
snapshot/2025-11-14T1512/parser-fragility
```

Properties:

* No counters
* Perfect chronological sorting
* Clean, readable
* Machine-parsable
* Unique without extra tokens

---

### **3.2 Snapshot Contents**

Inside a snapshot branch:

```
/
├── research/
│   └── 2025-11-14T0930/
│       └── security-audit/
│           ├── notes.md
│           ├── poc_*.*
│           ├── data/* (optional)
│           ├── meta.json
│           └── embedding.bin (optional)
└── full project code tree
```

Notes:

* All research artifacts live under `research/`
* Snapshot metadata lives in `meta.json`
* Embeddings are optional and stored separately

---

### **3.3 Snapshot Types / Modes**

The CLI must support modes:

| Mode               | Description                                  |
| ------------------ | -------------------------------------------- |
| **full** (default) | Full code tree + research artifacts          |
| **research-only**  | Only `research/` + reference commit hash     |
| **diff**           | Store a patch + research/ + reference commit |
| **poc**            | Only POC files + reference commit            |

This enables flexibility for large repos or shallow snapshots.

---

# **4. Metadata Specification (`meta.json`)**

### **Schema:**

```json
{
  "created_at": "2025-11-14T09:30:12Z",
  "topic": "security-audit",
  "root": "snapshot/2025-11-14T0930/security-audit",
  "mode": "full",
  "related_branch": "feature/security-hardening",
  "main_commit": "abc123",
  "tags": ["security", "audit", "investigation"],
  "embedding": "embedding.bin",
  "notes": "Optional free-form summary"
}
```

### Metadata Guarantees:

* Immutable after creation
* Standardized for agentic tools
* Allows timeline correlation
* Allows semantic search

---

# **5. Embeddings via Ollama (Local)**

Yes — **Ollama can be used for fully local embeddings**.

### **Embedding Generation Process**

1. CLI calls:

   ```
   ollama embed -m nomic-embed-text:latest <file>
   ```
2. Output is binary or JSON vector (depending on model)
3. Store as `embedding.bin`
4. Add reference to `meta.json`

### **Benefits**

* No external API
* Private / offline
* Fast
* Standardized vector format
* Perfect for semantic search in snapshot history

### **Supported Models**

* `nomic-embed-text` (best default)
* `all-minilm` variants
* Future LLM embeddings (pluggable)

### **Optional: Embedding Index**

Snapshots can be indexed in:

* SQLite + `sqlite-vss` (ideal)
* or a simple directory-based JSON store

---

# **6. CLI Specification (`snapshot`)**

### **6.1 Core Commands**

#### **snapshot save**

Create a snapshot.

```
snapshot save "pnpm migration"
```

Options:

```
--topic          override topic slug
--mode           full | diff | research-only | poc
--include        extra files
--tag            add metadata tags
--no-embed       skip embedding generation
```

---

#### **snapshot list**

List all snapshots.

```
snapshot list
snapshot list --topic security
snapshot list --today
snapshot list --since 2025-10-01
```

---

#### **snapshot open**

Create a worktree and open the snapshot:

```
snapshot open 2025-11-14T0930 security-audit
```

Equivalent to:

```
git worktree add ../snap-0930 snapshot/2025-11-14T0930/security-audit
```

---

#### **snapshot meta**

Show metadata.

---

#### **snapshot search**

Semantic search using embeddings.

```
snapshot search "parser fragility risks"
snapshot search --topic security "dependency vulnerabilities"
```

---

#### **snapshot prune**

Apply retention policy.

```
snapshot prune
```

Configurable in `~/.config/snapshot/config.toml`.

---

#### **snapshot archive**

Bundle snapshots for external storage.

```
snapshot archive 2024
```

---

### **6.2 CLI Implementation Language**

**Chosen language: Go**

Reasons:

* Single static binary
* Cross-platform
* Perfect Git subprocess handling
* Easy embedding of local LLM clients
* Good concurrency model (parallel embedding indexing)

---

# **7. Immutability Enforcement**

### **7.1 Git Hook**

`pre-commit`:

```bash
branch=$(git rev-parse --abbrev-ref HEAD)
if [[ $branch == snapshot/* ]]; then
    echo "Snapshots are immutable. Create a new snapshot instead."
    exit 1
fi
```

### **7.2 CLI Protection**

`snapshot save` refuses to overwrite existing snapshot branches.

### **7.3 Metadata Lock**

Store root-tree hash in `meta.json` to detect tampering.

---

# **8. Retention Policy**

Default:

* Keep snapshots for last 90 days
* Keep snapshots with `--tag important` indefinitely
* Prune others after expiry

Configurable via:

```
~/.config/snapshot/config.toml
```

Sample:

```toml
[retention]
days = 180
preserve_tags = ["security", "architecture"]
```

---

# **9. Agentic Integration**

Agents can:

* read metadata
* scan snapshots
* diff snapshots
* generate summaries
* semantically search embeddings
* reconstruct decision timelines

This makes snapshots a **machine-readable knowledge graph over time**.

Optional future enhancements:

* automatic tagging
* automated snapshot summaries
* similarity-based clustering
* agent-driven snapshot recommendations

---

# **10. Risks & Mitigations**

| Risk               | Mitigation                             |
| ------------------ | -------------------------------------- |
| Branch clutter     | CLI filtering + pruning + namespacing  |
| Unintended commits | Git hooks + CLI warning                |
| Repo bloat         | Snapshot modes + `.gitignore` guidance |
| Embedding cost     | Optional + cached embeddings           |
| Human misuse       | Strict namespace + tooling             |

---

# **11. Implementation Roadmap**

### **Phase 1: Baseline**

* Branch naming
* `save`, `list`, `open`
* Default full snapshots
* Metadata generation
* Git hook for immutability

### **Phase 2: Productivity**

* Diff/research-only modes
* Prune
* Archive
* Topic filtering

### **Phase 3: Agentic / Embedding**

* Ollama embeddings
* Snapshot semantic search
* SQLite index

---

# **12. Deliverables**

* Go CLI binary (`snapshot`)
* Git hooks bundle
* Default configuration file
* `research/` directory conventions
* Design docs + examples

modified line
