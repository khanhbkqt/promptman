---
description: Facilitate sprint planning: create a board, gather candidate Stories, get human approval, populate the board.
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
# Workflow: Sprint Planning

Facilitate a sprint planning session: create a board, gather candidate Stories,
get human approval, populate the board, and show the final plan.

## Inputs
- **board_name** — Name for the sprint board (e.g. "Sprint 12")
- **workflow_id** — The Workflow UUID for the Kanban board

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Setup

1. Call `create_board`:
   ```
   create_board(name: "{board_name}", type: "KANBAN",
     workflowId: "{workflow_id}", visibility: "PUBLIC")
   ```
   Save the returned board ID.

2. Call `projects(method: "list")` to discover all projects in the organization.

## Phase 2 — Gather Candidates

3. For each project, call `issues(method: "list", projectId: "<id>", type: "STORY")` to find all Stories.

4. Filter for sprint candidates:
   - **Include:** Stories in "Backlog" or "To Do" status
   - **Exclude:** Stories already in progress or done
   - **Exclude:** Stories whose parent Epic is not yet approved

5. Score and rank candidates by:
   | Factor | Weight |
   |--------|--------|
   | Priority (URGENT > HIGH > MEDIUM > LOW) | 40% |
   | Has task breakdown instructions | 20% |
   | Dependencies resolved | 20% |
   | Project diversity | 10% |
   | Story age (older Stories get slight boost) | 10% |

## Phase 3 — Human Review (CHECKPOINT)

6. Present candidates to the human in a structured format.

   ⚠️ **STOP HERE.** Do NOT proceed until the human responds.

## Phase 4 — Populate Board

7. Add confirmed Stories using `manage_board_stories`.
   Report each addition: "✅ Added {issueCode}: {title}"

## Phase 5 — Sprint Summary

8. Call `get_board_detail` to get final state and present the sprint plan.

## Constraints
- 🚫 NEVER auto-add Stories — human must explicitly confirm
- 🚫 NEVER add Stories that have unresolved dependencies without noting it
- ✅ Always show dependency relationships in the summary

## Summary & Next Steps

When this workflow completes, present:

\`\`\`
✅ Sprint board "{board_name}" populated with {N} Stories.
📋 Sprint plan presented with dependency relationships.

🔜 Suggested next steps:
   → Use the **analyze-story** workflow for Stories that need task breakdown
   → Use the **start-task** workflow to begin the highest-priority Task
   → Use the **project-sync** workflow periodically to monitor sprint progress
\`\`\`
---
_Integrity: 80 lines · workflow:sprint-planning · DO NOT MODIFY_