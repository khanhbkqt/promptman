---
name: vibepm-task-execution
description: |
  Execute work in VibePM — pick up Stories, break them into Tasks, implement
  code, handle dependencies, complete Tasks, and handle rework after review.
  Use when "start task", "pick up story", "implement", "complete task",
  "break down story", "create tasks", "rework", or "how to deliver work".
---

# VibePM Task Execution

This skill covers the full execution lifecycle from picking up work to delivering it.
For Member agents primarily, but useful for anyone understanding the delivery flow.

## The Execution Model

```
Pick Up Work → Analyze Story → Create Tasks → Start Task → Implement → Complete → [Rework]
```

**Key principle:** You define HOW to implement. The PM already defined WHAT in the Story.

> 📋 **Use the dedicated workflows for each phase:**
> - **Received a Story?** → Run `/pm-analyze-story` for full document discovery and Task breakdown
> - **Starting a Task?** → Run `/pm-start-task` to load all context before writing any code


## Step 0: Check Notifications

Before picking up new work, check for unread notifications:

```
notification(method: "list", isRead: false, limit: 10)
→ Shows unread notifications: assignments, review comments, status changes
```

Act on relevant ones first:
- **ASSIGNED** → New work assigned to you. Check priority.
- **COMMENTED** → Review feedback or questions. Respond before starting new work.
- **STATUS_CHANGED** → A task you care about moved. Update your mental model.
- **MENTIONED** → Someone referenced you. Read the context and respond.

After processing, mark as read:
```
notification(method: "read_all")
```

## Step 1: Pick Up Work

Find your assigned work:

```
issues(method: "list", projectId: "PRJ", mine: true)
→ Returns your assigned issues with status, type, and priority
```

Or use the resource:
```
Or use the **my-tasks** MCP resource for the project.
```

## Step 2: Analyze a Story

When assigned a Story (not a Task), you must analyze it before coding:

```
1. issues(method: "get", projectId: "PRJ", issueId: "PRJ-51")
   → Returns:
     - Story description with acceptance criteria and suggested breakdown
     - Parent Epic context (broader feature goals)
     - Sibling Stories (what's being built in parallel)
     - Comments (clarifications from PM)
     - Linked Documents (architecture, API docs)

2. Read the linked documents:
   docs(method: "get", projectId: "PRJ", documentId: "<linked-doc-id>")

3. Check the PM's suggested task breakdown in the Story description:
   - Does it make technical sense?
   - Are there hidden complexities?
   - Missing tasks (migrations, config, tests)?

4. Write your implementation plan as a comment:
   issues(method: "comment", projectId: "PRJ", issueId: "PRJ-51",
     content: "## Implementation Plan\n\n1. **Search query builder** — will use TypeORM QueryBuilder...\n2. **REST endpoint** — GET /api/search with Zod validation...\n3. **Integration tests** — seed test data, test pagination...")

5. STOP — Wait for human approval before creating Tasks.
```

## Step 3: Break Down into Tasks

After human approves your plan, create Tasks:

```
projects(method: "get", projectId: "PRJ")
→ Find the Backlog status ID: "s1"

issues(method: "create",
  projectId: "PRJ",
  title: "Implement search query builder",
  type: "TASK",
  statusId: "s1",         ← Backlog
  parentId: "<story-id>",  ← Under the Story
  priority: "HIGH",
  description: "Build the full-text search query layer using TypeORM QueryBuilder..."
)

issues(method: "create",
  projectId: "PRJ",
  title: "Add search REST endpoint",
  type: "TASK",
  statusId: "s1",
  parentId: "<story-id>",
  priority: "HIGH",
  description: "GET /api/search endpoint with query, pagination, and error handling..."
)
```

**Sizing guidelines:**
- 2-4 Tasks per Story (ideal)
- Each Task = one focused work session
- Include tests WITH implementation (not as a separate task)
- Don't create "research" tasks — do research during analysis

## Step 4: Start a Task

Before writing any code:

```
1. issues(method: "get", projectId: "PRJ", issueId: "PRJ-52")
   → Read the Task + its parent Story's acceptance criteria

2. Check dependencies:
   → Are sibling Tasks that yours depends on still incomplete?
   → If blocked: issues(method: "comment") explaining the blocker, then STOP

3. Update status:
   issues(method: "update",projectId: "PRJ", issueId: "PRJ-52",
     statusId: "<ai-working-or-in-progress-id>")

4. Create a feature branch:
   → Pattern: feat/{issue-code}-short-slug
   → Example: feat/PRJ-52-search-query-builder

5. Present your plan to the human:
   "## 🚀 Starting: PRJ-52 — Search query builder
    **What I'll implement:** Full-text search using TypeORM QueryBuilder
    **Acceptance criteria I'll satisfy:** [list from parent Story]
    **Files I expect to modify:** src/search/query-builder.ts, src/search/index.ts
    **Branch:** feat/PRJ-52-search-query-builder"
```

## Step 5: Complete a Task

After implementation, don't just stop:

```
1. Self-review against parent Story's acceptance criteria:
   issues(method: "get", projectId: "PRJ", issueId: "PRJ-52")

   Check each criterion:
   | Criterion | Met? | Evidence |
   |-----------|------|----------|
   | Returns matching results | ✅ | Tested with sample data |
   | Supports pagination | ✅ | limit/offset params work |

2. Run tests and verify commits reference the issue code:
   → All commits should be: feat(PRJ-52): description

3. Post implementation summary:
   issues(method: "comment", projectId: "PRJ", issueId: "PRJ-52",
     content: "## ✅ Implementation Complete\n\n**Branch:** feat/PRJ-52-search-query-builder\n\n### Changes Made\n- Added QueryBuilder in src/search/\n- Added unit tests for edge cases\n\n### Acceptance Criteria\n- [x] Full-text search works — tested with 1000 records\n- [x] Pagination — limit/offset params functional\n\n### Tests\n- Status: All passing (12 new tests)\n\n### Documentation Impact\n- API Reference doc needs update (new endpoint)")

4. Move to In Review:
   issues(method: "update",projectId: "PRJ", issueId: "PRJ-52",
     statusId: "<in-review-status-id>")
```

## Step 6: Handle Rework

When a reviewer sends your task back to "In Progress":

```
1. issues(method: "get", projectId: "PRJ", issueId: "PRJ-52")
   → Read the review comment — distinguish "Must Fix" from "Should Fix"

2. If feedback is unclear:
   issues(method: "comment", projectId: "PRJ", issueId: "PRJ-52",
     content: "Clarification needed: regarding the SQL injection concern, did you mean...")
   → STOP and wait

3. Address all "Must Fix" items with focused changes

4. Re-run tests

5. Post rework summary:
   issues(method: "comment", projectId: "PRJ", issueId: "PRJ-52",
     content: "## 🔄 Rework Complete\n\n### Changes\n- Added input sanitization (parameterized queries)\n- Added max length validation (500 chars)\n- Added SQL injection test\n\n### Review Items Addressed\n- [x] Must Fix: SQL injection — fixed with parameterized queries\n- [x] Must Fix: Input validation — added max length\n\nReady for re-review.")

6. Move back to In Review:
   issues(method: "update",projectId: "PRJ", issueId: "PRJ-52",
     statusId: "<in-review-status-id>")
```

## Use Case: End-to-End Story Delivery

Complete flow for delivering Story PRJ-51 "Search API Endpoint":

```
1. issues(method: "list", projectId: "PRJ", mine: true)
   → See PRJ-51 assigned to you

2. issues(method: "get", projectId: "PRJ", issueId: "PRJ-51")
   → Read Story description, acceptance criteria, linked docs

3. docs(method: "get", projectId: "PRJ", documentId: "api-reference")
   → Understand existing API patterns

4. issues(method: "comment", projectId: "PRJ", issueId: "PRJ-51",
     content: "## Implementation Plan\n...")
   → CHECKPOINT: Wait for human approval

5. issues(method: "create",projectId: "PRJ", title: "Search query builder", type: "TASK", ...)
   issues(method: "create",projectId: "PRJ", title: "Search endpoint", type: "TASK", ...)
   → Create Tasks under the Story

6. For each Task: start → implement → complete (steps 4-5 above)

7. After all Tasks complete:
   → Check docs: docs(method: "list") → docs(method: "update") if needed
   → Link updated docs: issues(method: "update",linkDocumentIds: [...])
   → Leave audit trail comment
```

## Common Execution Mistakes

1. **Coding without reading the parent Story** — Your Task may not have full acceptance criteria. Always read the parent.
2. **Skipping dependency checks** — If a sibling Task must finish first, you'll waste work.
3. **No implementation summary** — Reviewers need evidence. Always post the structured comment.
4. **Giant commits without issue codes** — Use `feat(PRJ-52): description` for every commit.
5. **Moving issues to Done** — You can't. The backend will return 403. Move to "In Review" and stop.
6. **Ignoring notifications** — Review feedback or new assignments may be waiting. Always check notifications first.
