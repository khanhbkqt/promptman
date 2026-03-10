---
description: Pre-work checklist: gather full context, understand requirements, check dependencies, set status, create branch.
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
# Workflow: Start Task

Comprehensive pre-work checklist: gather full context from the issue hierarchy,
understand requirements, check dependencies, set status, create branch, and confirm readiness.

## Inputs
- **project_id** — The VibePM project ID or code
- **issue_id** — The VibePM issue ID or code to start

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 0 — Check Notifications

Before gathering context, check for unread notifications relevant to this task:

1. Call \`notification(method: "list", isRead: false, limit: 10)\`
2. Look for notifications related to your task (comments, status changes)
3. If review feedback exists on this task → read and address it first
4. Mark processed notifications as read: \`notification(method: "read", notificationId: "<id>")\`

## Phase 1 — Understand Context

1. Call `projects(method: "get", projectId: "{project_id}")` to retrieve:
   - Project statuses → find "In Progress" status ID
   - Project code → for branch naming

2. Call `issues(method: "get", projectId: "{project_id}", issueId: "{issue_id}")` to retrieve:
   - Issue details (title, type, description, priority)
   - Parent issue (Story or Epic) with its description
   - Sibling issues (other Tasks under the same Story)
   - All existing comments

3. **If the parent is a Story**, read its description carefully for:
   - **Acceptance criteria** → these define "done" for your work
   - **Suggested task breakdown** → check if your task matches one of these
   - **Technical notes** → implementation guidance from the PM
   - **Dependencies** → are prerequisite Stories complete?
   - **Out of scope** → what you should NOT do

4. **If the issue itself is a Story** (no Tasks exist yet), you need to break it down:
   - Read the Story's suggested task breakdown
   - Create Tasks under this Story using `issues(method: "create", ...)`
   - Then start working on the first Task (re-read this workflow with the Task ID)

## Phase 2 — Check Dependencies

5. Review sibling issues (other Tasks under the same Story):
   - Are any Tasks that yours depends on still incomplete?
   - If blocked by a dependency:
     a. Call `issues(method: "comment", ...)` on your issue explaining the blocker
     b. STOP — inform the human: "This task is blocked by {dependency}."

6. Call `issues(method: "list", projectId: "{project_id}", mine: true)` to see your full workload.

## Phase 3 — Set Status, Assign & Branch

7. Call `issues(method: "update", ...)` to mark the task active and claim ownership:
   - Set `assignToMe: true` to assign this task to yourself
   - Use "In Progress" status
   - Example: `issues(method: "update", projectId: "{project_id}", issueId: "{issue_id}", statusId: "<status_id>", assignToMe: true)`

8. Create a feature branch:
   - Naming pattern: `feat/{issue-code}-short-slug`
   - Example: `feat/PRJ-42-search-api`

## Phase 4 — Confirm Readiness

9. Present your understanding to confirm before coding:

   ```markdown
   ## 🚀 Starting: {issueCode} — {title}

   **What I'll implement:** <1-2 sentence summary>
   **Acceptance criteria I'll satisfy:** <list criteria>
   **Key technical approach:** <how you plan to implement>
   **Files I expect to modify:** <file/module list>
   **Branch:** feat/{issue-code}-{slug}
   **Dependencies:** {resolved / blocked by X}
   ```

   ⚠️ Wait for human confirmation if the task is ambiguous or complex.

## Git Convention Rules

- **Branch naming:** \`feat/{issue-code}-short-slug\` (e.g. \`feat/PRJ-42-search-api\`)
- **Commit format:** \`feat({issue-code}): description\` (e.g. \`feat(PRJ-42): add search endpoint\`)
- **ALL commits** on this branch MUST reference the issue code — no exceptions
- **Commit messages** should be concise but descriptive of the change
- Use \`fix({issue-code}): ...\` for bug fixes within the task
- Use \`refactor({issue-code}): ...\` for refactoring within the task

## Constraints
- 🚫 NEVER start coding without reading the parent Story's acceptance criteria
- 🚫 NEVER skip dependency checks
- 🚫 NEVER make commits without the issue code reference
- ✅ ALWAYS reference issue code in all commits: \`feat({code}): description\`
- ✅ ALWAYS update status before starting work
- ✅ ALWAYS create the feature branch before writing any code

## Summary & Next Steps

When this workflow completes, present:

\`\`\`
✅ Task {issueCode} started — status set to In Progress.
🌿 Branch created: feat/{issue-code}-{slug}
📋 Context gathered, readiness confirmed.

🔜 Suggested next steps:
   → Implement the task following the acceptance criteria
   → When done, use the **complete-task** workflow to verify and hand off for review
   → Remember: all commits must use \`feat({issueCode}): description\` format
\`\`\`
---
_Integrity: 120 lines · workflow:start-task · DO NOT MODIFY_