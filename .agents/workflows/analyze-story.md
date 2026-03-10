---
description: 
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
# Workflow: Analyze Story

Perform deep technical analysis of a Story: review all context and docs, write an
implementation plan as a comment, get human approval, then create appropriately-sized Tasks.

## Inputs
- **project_id** — The VibePM project ID or code
- **issue_id** — The Story issue ID to analyze

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Deep Context Gathering

1. Call `issues(method: "get", projectId: "{project_id}", issueId: "{issue_id}")` to retrieve:
   - Story details, description, acceptance criteria
   - Parent Epic context (broader feature goals)
   - Sibling Stories (what's being built in parallel)
   - Existing comments (clarifications, decisions)
   - Linked documents (read these first!)

2. Call `projects(method: "get", projectId: "{project_id}")` to get:
   - Project statuses → find "Backlog" status ID for task creation
   - Project code → for naming conventions

3. Call `docs(method: "list", projectId: "{project_id}")` and read relevant documents:
   - **Architecture docs** — understand the system structure
   - **API docs** — existing endpoints and contracts
   - **Data model docs** — database schema and relationships

   You MUST read at least the architecture doc if one exists, as well as any documents linked to the Story or its Epic.

4. Check the Story's "Suggested Task Breakdown" from the PM:
   - Does it make sense from a technical standpoint?
   - Are there hidden complexities the PM may have missed?
   - Are there additional tasks needed (migrations, tests, config)?

## Phase 2 — Technical Analysis

5. For each piece of work identified, analyze:
   - **Affected files/modules** — which parts of the codebase change
   - **Approach** — how to implement (pattern, library, algorithm)
   - **Risks** — what could go wrong
   - **Unknowns** — what needs investigation during implementation

6. Identify cross-cutting concerns:
   - Database migrations needed?
   - API contract changes?
   - Configuration or environment variable changes?
   - Impact on existing tests?
   - Security implications?

7. **Task sizing guidance** — each Task should be:
   - A **meaningful unit of work** — not "add a field" but "implement search endpoint"
   - **2-4 tasks per Story** is ideal
   - Each task should be completable in one focused session
   - Tasks should have clear start and end conditions

   **Anti-patterns:**
   - ❌ One task per file change (too granular)
   - ❌ "Write tests" as a separate task (tests belong with implementation)
   - ❌ "Research" as a task (do research NOW in analysis)
   - ❌ One giant task for the whole Story

## Phase 3 — Implementation Plan Comment (CHECKPOINT)

8. Post your implementation plan as a comment on the Story using `issues(method: "comment", ...)`.

9. Present to the human and **STOP.** Wait for human approval before creating Tasks.

## Phase 4 — Create Tasks

10. After human approval, create Tasks under the Story using `issues(method: "create", ...)` with type "TASK".

11. Report the created Tasks and suggest using the **start-task** workflow for the first task.

## Constraints
- 🚫 NEVER create Tasks without reading project documentation first
- 🚫 NEVER create Tasks without human approval of the implementation plan
- 🚫 NEVER create more than 5 Tasks per Story
- ✅ ALWAYS check for database and API impact
- ✅ ALWAYS include test expectations within each task
---
_Integrity: 88 lines · workflow:analyze-story · DO NOT MODIFY_