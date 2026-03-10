---
description: Facilitate a structured brainstorming session: gather project context, diverge on ideas, converge into a clear document, and get human sign-off — all BEFORE creating any issues.
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
# Workflow: Brainstorm

Facilitate a structured brainstorming session: gather project context, diverge on ideas,
converge into a clear document, and get human sign-off — all BEFORE creating any issues.

## Inputs
- **project_id** — The VibePM project ID or code
- **topic** — Topic or area to brainstorm (e.g. "user onboarding flow", "monetization strategy")

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Gather Context

1. Call `projects(method: "get", projectId: "{project_id}")` to understand the project's domain.

2. Call `docs(method: "list", projectId: "{project_id}")` and scan for documents that provide context:
   - Product vision, PRDs, architecture docs
   - Previous brainstorm outputs
   - Existing feature specs that relate to the topic

3. Read the most relevant documents using `docs(method: "get", ...)` to build mental context.
   - Pay attention to existing features, constraints, and technical boundaries.

4. Call `issues(method: "list", projectId: "{project_id}", type: "EPIC")` to see what's already planned.
   - Identify overlaps or gaps relevant to the brainstorm topic.

## Phase 2 — Divergent Ideation

5. Generate ideas freely. For each idea, capture:
   - **Title** — short, descriptive name
   - **Description** — what it is and why it matters
   - **User value** — who benefits and how
   - **Rough complexity** — Small / Medium / Large / Unknown
   - **Dependencies** — what needs to exist first

6. Aim for **8-15 raw ideas** covering different angles:
   - Obvious improvements
   - Bold/ambitious features
   - Quick wins
   - Infrastructure/foundation work
   - User experience refinements

   **Anti-patterns to avoid:**
   - ❌ Self-censoring — include wild ideas, the human will filter
   - ❌ Jumping to solutions — describe the outcome, not implementation
   - ❌ Repeating existing features — check what's already planned

## Phase 3 — Convergent Structuring

7. Group ideas into logical themes (3-5 themes max).

8. For each theme, assess:
   - **Priority signal** — HIGH (core to product) / MEDIUM (valuable) / LOW (nice to have)
   - **Feasibility** — Ready (can start now) / Needs research / Blocked by X
   - **Relationship to existing work** — extends Epic X / standalone / prerequisite for Y

## Phase 4 — Capture as Document (CHECKPOINT)

9. Create a VibePM document with the brainstorm output:

   ```
   docs(method: "create",
     projectId: "{project_id}",
     title: "Brainstorm: {topic}",
     content: "<structured content — see template below>",
     parentId: "<brainstorming-master-doc-id>"
   )
   ```

   **Document Template:**
   ```markdown
   # Brainstorm: {topic}

   **Date:** <today>
   **Status:** Draft — awaiting review

   ## Context
   <Brief summary of the project state and why this brainstorm was initiated>

   ## Theme 1: <Theme Name>
   **Priority:** HIGH / MEDIUM / LOW

   ### Idea 1.1: <Title>
   - **What:** <description>
   - **User value:** <who benefits>
   - **Complexity:** Small / Medium / Large
   - **Dependencies:** <list or "None">

   ## Recommendations
   - **Start with:** <top 2-3 ideas to move forward>
   - **Needs research:** <ideas that need more exploration>
   - **Park for later:** <ideas to revisit in future>

   ## Next Steps
   - [ ] Human reviews and selects ideas to pursue
   - [ ] Link this document to relevant Epics using \`issues(method: "update", ...)\` (\`linkDocumentIds\`)
   - [ ] Use the **plan-scope** workflow to define a milestone
   - [ ] Use the **plan-feature** workflow for each approved feature
   ```

10. Present the document summary to the human and **STOP.** Wait for human feedback.

## Constraints
- 🚫 Do NOT create Epics, Stories, or Tasks during brainstorming
- 🚫 Do NOT skip the human review checkpoint
- ✅ Always capture output as a VibePM document
- ✅ Always include recommendations with clear priority signals
---
_Integrity: 114 lines · workflow:brainstorm · DO NOT MODIFY_
