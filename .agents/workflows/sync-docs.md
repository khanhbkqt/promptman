---
description: After completing code changes, systematically identify and update all affected project documents. Creates an audit trail of what was updated.
---

> [!CAUTION]
> **VERBATIM COPY REQUIRED** — This workflow MUST be copied exactly as-is.
> Do NOT summarize, compress, rephrase, or omit any part.
> Do NOT rewrite steps in your own words.
> Every line, bullet, code block, and table must appear in the output unchanged.
> Violation = broken workflow that will fail at runtime.
> [!TIP]
> **Tool discovery:** Unsure about a tool's parameters? Call any tool with `method: "help"` to list commands,
> or `method: "help", command: "<name>"` to see the full parameter table for a specific command.
# Workflow: Sync Docs

After completing code changes, systematically identify and update all affected project documents.
Creates an audit trail of what was updated.

## Inputs
- **project_id** — The VibePM project ID or code
- **issue_id** — The issue that was just completed
- **changed_files** — (optional) Comma-separated list of changed files

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Gather Context

1. Call `issues(method: "get", projectId: "{project_id}", issueId: "{issue_id}")` to understand:
   - What was implemented (read the description and comments)
   - Which parent Story/Epic this belongs to (broader context)
   - Any child issues that were part of this work

2. Call `docs(method: "list", projectId: "{project_id}")` to get the full document tree.

3. Identify the scope of changes:
   - If changed_files provided, use them directly.
   - Otherwise, infer from the issue description and implementation comments what areas of code were touched.

## Phase 2 — Analyze Impact

4. For each document, evaluate whether it needs updating by checking:

   | Document Type | Update Triggers |
   |--------------|----------------|
   | API docs | New/changed endpoints, request/response schemas, auth changes |
   | Architecture docs | New modules, changed dependencies, data flow changes |
   | Setup/install guides | New env vars, dependencies, build steps |
   | Feature descriptions | New capabilities, changed behavior, removed features |
   | Database docs | Schema changes, new tables, migration notes |

5. Classify each document into one of three buckets:
   - **NEEDS UPDATE** — content is directly affected by the changes
   - **NO UPDATE** — content is not affected
   - **GAP** — a document should exist but doesn't (e.g., new API endpoint with no docs)

## Phase 3 — Update Documents

6. For each document marked NEEDS UPDATE:
   a. Call `docs(method: "get", projectId: "{project_id}", documentId: "{doc-id}")` to read current content.
   b. Identify the specific sections that need changes.
   c. Call `docs(method: "update", projectId: "{project_id}", documentId: "{doc-id}", content: "{updated-content}")`
   - Preserve existing content structure — only modify affected sections.
   - Add a "Last updated" note if the document has one.

7. For each GAP identified:
   - Note it for the audit comment but do NOT create the document unless it's critical.
   - If critical (e.g., a new public API with zero docs), call `docs(method: "create", ...)` to create a stub.

## Phase 4 — Audit Trail

8. Call `issues(method: "comment", projectId: "{project_id}", issueId: "{issue_id}")` with body:
   ```markdown
   ## 📄 Documentation Sync

   **Updated:**
   - {doc title} — {what changed}

   **No update needed:**
   - {doc title} — {reason}

   **Gaps flagged:**
   - {missing doc description} — {why it's needed}

   **Summary:** {N} documents updated, {M} gaps flagged.
   ```

## Error Recovery
- If `docs(method: "get", ...)` fails for a document, skip it and note in the audit.
- If `docs(method: "update", ...)` fails, retry once. If it fails again, note the failure in the audit comment.
---
_Integrity: 83 lines · workflow:sync-docs · DO NOT MODIFY_
