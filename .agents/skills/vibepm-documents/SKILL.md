---
name: vibepm-documents
description: |
  Master VibePM's document system — scanning briefs, reading full docs,
  creating knowledge, and linking documents to issues for traceability.
  Use when "project documentation", "knowledge base", "list docs", "create doc",
  "link document", "update documentation", or "sync docs".
---

# VibePM Document Management

VibePM has a hierarchical document system that serves as the project knowledge base.
Documents are linked to issues for context and traceability.

## Document Structure

Each document has:
- **title** — Human-readable name
- **brief** — 1-2 sentence summary (shown in listings — critical for scanning)
- **content** — Full document body (plain text, auto-converted to rich text)
- **slug** — URL-friendly identifier (auto-generated from title)
- **parentId** — Optional parent document for nesting (folder structure)

## Reading Documents: The Brief-First Pattern

**Always scan briefs before reading full content.** This saves tokens and context.

```
Step 1: docs(method: "list", projectId: "PRJ")
→ Returns: [
    { id: "d1", title: "Architecture", brief: "System architecture, tech stack, deployment topology", parentId: null },
    { id: "d2", title: "API Reference", brief: "REST endpoints, request/response schemas", parentId: "d1" },
    { id: "d3", title: "Data Model", brief: "Database entities, relationships, migration history", parentId: "d1" },
    { id: "d4", title: "Getting Started", brief: "Local dev setup, environment variables, first run", parentId: null }
  ]

Step 2: Decide which to read based on your task
→ Working on a new API endpoint? Read "API Reference" and "Data Model"
→ docs(method: "list") showed brief tells you enough about "Getting Started" — skip it

Step 3: docs(method: "get", projectId: "PRJ", documentId: "d2")
→ Returns full content of "API Reference"
```

## Creating Documents

When creating docs, **always include a brief**.

> Document `content` accepts **Markdown**. The backend converts it to Tiptap JSON.
> Supports: `# headings`, `**bold**`, `*italic*`, `` `code` ``, `- lists`, `1. ordered`,
> `- [ ] tasks`, `> blockquotes`, code blocks, `---` rules.
> Read the `vibepm-content-format` skill for full examples:
> `read_resource("vibepm://agent-skills/vibepm-content-format")`

```
docs(method: "create",
  projectId: "PRJ",
  title: "Search API Design",
  brief: "Full-text search endpoint design — query syntax, pagination, and filtering",
  content: "# Search API Design\n\n## Endpoint\n\n`POST /api/v1/search`\n\n## Request Fields\n\n- `q` — search query (required)\n- `limit` — max results, default 20"
)
```

### Nesting Documents

Use `parentId` to organize documents under folders:

```
docs(method: "create",
  projectId: "PRJ",
  title: "Search Performance",
  brief: "Benchmarks and optimization notes for the search subsystem",
  content: "...",
  parentId: "d2"  ← nests under "API Reference"
)
```

## Updating Documents

`docs(method: "update")` accepts partial updates — pass only what changed:

```
docs(method: "update",
  projectId: "PRJ",
  documentId: "d2",
  brief: "REST endpoints including new search API — request/response schemas",
  content: "...updated content with new sections..."
)
```

**Always update the `brief` if the document's scope has changed.**

## Linking Documents to Issues

Documents can be linked to issues for context. This is bidirectional — the issue
shows linked docs, and `issues(method: "get")` returns them.

```
issues(method: "update",
  projectId: "PRJ",
  issueId: "PRJ-42",
  linkDocumentIds: ["d2", "d3"]    ← link API Reference and Data Model
)
```

To unlink:
```
issues(method: "update",
  projectId: "PRJ",
  issueId: "PRJ-42",
  unlinkDocumentIds: ["d3"]        ← remove Data Model link
)
```

## Use Case: Update Docs After Code Changes

After implementing a feature, documentation may be stale. Follow this pattern:

```
1. docs(method: "list", projectId: "PRJ")
   → Scan briefs to find potentially affected docs

2. Identify which docs your code changes impacted:
   - New API endpoint? → Update API Reference
   - New database table? → Update Data Model
   - New env variable? → Update Getting Started

3. docs(method: "get", projectId: "PRJ", documentId: "<affected-doc>")
   → Read current content

4. docs(method: "update",projectId: "PRJ", documentId: "<affected-doc>",
     content: "<updated content>",
     brief: "<updated brief if scope changed>")

5. issues(method: "update",projectId: "PRJ", issueId: "PRJ-42",
     linkDocumentIds: ["<updated-doc-id>"])
   → Link the updated doc to the issue for traceability

6. issues(method: "comment",projectId: "PRJ", issueId: "PRJ-42",
     content: "Updated API Reference doc to include new search endpoint.")
   → Leave audit trail
```

## Use Case: Research Before Planning

Before planning a new feature, gather existing knowledge:

```
1. docs(method: "list", projectId: "PRJ")
   → Scan all briefs

2. Read docs relevant to the feature area:
   docs(method: "get", projectId: "PRJ", documentId: "architecture")
   docs(method: "get", projectId: "PRJ", documentId: "data-model")

3. Present findings to the human:
   "I found 3 relevant docs: Architecture, Data Model, and API Reference.
    The architecture doc shows we use X pattern. Should I proceed with planning?"

4. Link relevant docs to the new Epic when created:
   issues(method: "update",projectId: "PRJ", issueId: "PRJ-50",
     linkDocumentIds: ["d1", "d2", "d3"])
```

## Document Conventions

VibePM recommends organizing documents **by type** using a hierarchical folder structure.
Each "folder" is a parent document; children nest under it via `parentId`.

### Recommended Structure

```
📁 Project Documents
├── 📁 Specs           ← Feature specifications and requirements
├── 📁 Design          ← Architecture decisions, API design, schemas
├── 📁 Architecture    ← System-level documentation, tech stack
├── 📁 Guidelines      ← Team conventions, checklists, coding standards
│   ├── Coding Guidelines    ★ Read by agents during Quality Gate
│   └── PR Checklist         ★ Used during review workflow
└── 📁 Meeting Notes   ← Brainstorm results, sprint planning notes
```

### Setting Up Document Structure

For a **new project**, create the top-level folders first:

```
docs(method: "create",projectId: "PRJ", title: "Specs", brief: "Feature specifications and requirements")
docs(method: "create",projectId: "PRJ", title: "Design", brief: "Architecture decisions, API design, database schemas")
docs(method: "create",projectId: "PRJ", title: "Architecture", brief: "System-level documentation, tech stack, deployment")
docs(method: "create",projectId: "PRJ", title: "Guidelines", brief: "Team conventions, checklists, and coding standards")
docs(method: "create",projectId: "PRJ", title: "Meeting Notes", brief: "Brainstorm results, sprint planning notes")
```

Then nest documents under the appropriate parent:

```
docs(method: "create",
  projectId: "PRJ",
  title: "Coding Guidelines",
  brief: "Code quality standards, naming conventions, and review checklist",
  content: "## Coding Guidelines\n\n...",
  parentId: "<guidelines-folder-id>"
)
```

### The Guidelines Folder

The `Guidelines/` folder is special — it contains team-defined documents that agents
read during the **complete-task** workflow (Quality Gate). Key documents:

- **Coding Guidelines** — Code quality standards, naming conventions, patterns to follow
- **PR Checklist** — Items to verify before submitting for review

Agents will search for these documents by scanning `docs(method: "list")` briefs, then reading
the full content via `docs(method: "get")` to self-review their work.
