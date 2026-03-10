---
name: vibepm-planning
description: |
  Plan features in VibePM — create Epics, write Story descriptions with
  acceptance criteria and task breakdowns, manage sprint boards, and track progress.
  Use when "plan feature", "create epic", "write stories", "sprint planning",
  "create board", "track progress", or "story template".
---

# VibePM Feature Planning

This skill covers how to plan features, write Stories, manage sprints, and track progress.
Primarily for PM agents, but useful for any agent needing planning context.

## The Planning Model

```
Brainstorm → Scope → Plan Feature → Sprint Planning → Track Progress
  (ideas)   (milestone) (Epic/Stories)    (board)         (monitor)
```

**Key principle:** PM defines WHAT (Epics + Stories). Members define HOW (Tasks).

## Planning a Feature: Step by Step

### Phase 1 — Discovery

Before creating anything, understand what already exists:

```
1. projects(method: "list")
   → Find the target project

2. projects(method: "get", projectId: "PRJ")
   → Get project statuses (you need the Backlog status ID)
   → Returns: { statuses: [{ id: "s1", name: "Backlog" }, ...] }

3. docs(method: "list", projectId: "PRJ")
   → Scan briefs for relevant architecture/API docs

4. docs(method: "get", projectId: "PRJ", documentId: "architecture")
   → Read relevant documents

5. issues(method: "list", projectId: "PRJ", type: "EPIC")
   → Check for existing related Epics (avoid duplicates)
```

Present findings to the human: _"I found these related docs and existing Epics.
Should I proceed with planning?"_

### Phase 2 — Create the Epic

```
issues(method: "create",
  projectId: "PRJ",
  title: "Full-text Search",
  type: "EPIC",
  statusId: "<backlog-status-id>",
  priority: "HIGH",
  description: "Add full-text search across all content types..."
)
→ Returns: { id: "epic-1", issueCode: "PRJ-50" }
```

Link relevant documents to the Epic:
```
issues(method: "update",
  projectId: "PRJ",
  issueId: "PRJ-50",
  linkDocumentIds: ["doc-arch", "doc-api"]
)
```

### Phase 3 — Create Stories with the Template

Each Story MUST include this markdown template in its description:

```markdown
## Overview
<What this Story delivers and why it matters>

## Acceptance Criteria
- [ ] <Specific, testable criterion>
- [ ] <Edge cases and error scenarios>
- [ ] <Performance requirements if any>

## Suggested Task Breakdown
1. **<Task title>** — <Brief description, estimated complexity, affected files>
2. **<Task title>** — <Brief description, estimated complexity, affected files>

## Technical Notes & Documents
- <Reference architecture docs, API patterns, libraries>
- <Link to relevant VibePM documents by ID>

## Dependencies
- <Other Story codes that must complete first, or "None">

## Out of Scope
- <What this Story explicitly does NOT cover>
```

**Example Story creation:**
```
issues(method: "create",
  projectId: "PRJ",
  title: "Search API Endpoint",
  type: "STORY",
  statusId: "<backlog-status-id>",
  priority: "HIGH",
  parentId: "epic-1",
  description: "## Overview\nAdd a full-text search API endpoint...\n\n## Acceptance Criteria\n- [ ] GET /api/search?q=term returns matching results\n- [ ] Supports pagination (limit/offset)\n- [ ] Returns results within 200ms for 10k records\n\n## Suggested Task Breakdown\n1. **Search query builder** — Build the query layer with TypeORM, ~medium complexity, affects src/search/\n2. **Search API controller** — REST endpoint with validation and pagination, ~medium, affects src/api/\n3. **Search integration tests** — E2E tests with test data seeding, ~low, affects test/\n\n## Technical Notes & Documents\n- See Architecture doc (id: doc-arch) for API patterns\n- Use existing pagination utility from src/common/\n\n## Dependencies\n- None\n\n## Out of Scope\n- Fuzzy matching (future Story)\n- Search UI (separate Story)"
)
```

**Create 2-4 Stories per Epic.** Each must be independently shippable.

## Sprint Planning

### Create a Board

```
create_board(
  name: "Sprint 5 — Mar 2026",
  type: "KANBAN",
  visibility: "PUBLIC",
  startDate: "2026-03-01",
  endDate: "2026-03-15"
)
→ Returns: { id: "board-1" }
```

### Add Stories to the Board

```
manage_board_stories(
  boardId: "board-1",
  projectId: "PRJ",
  issueId: "PRJ-51",    ← Story issue code
  action: "add"
)
```

**Only add Stories** (type STORY) — Tasks live under Stories, not on boards.

### Track Progress

```
get_board_detail(boardId: "board-1")
→ Returns: {
    board: { name: "Sprint 5", stories: [...] },
    progress: {
      overall: { total: 8, done: 3, inProgress: 2, percentage: 37.5 },
      byProject: { PRJ: { total: 5, done: 2 }, THN: { total: 3, done: 1 } }
    }
  }
```

## Monitoring Progress

Use these tools for ad-hoc tracking:

| Tool | Use for |
|------|---------|
| `get_board_detail(boardId)` | Sprint completion stats per project |
| `reports(method: "project_health", projectId: "PRJ")` | Check project health — spot bottlenecks |
| `issues(method: "list", projectId: "PRJ", type: "STORY")` | Find Stories needing attention |
| `issues(method: "list", projectId: "PRJ", statusId: "<in-review-id>")` | Issues waiting for review |

## Common Planning Mistakes

1. **Creating Tasks directly** — PMs create Epics + Stories only. Tasks are created by Members.
2. **Missing acceptance criteria** — Every Story needs testable criteria. "Make it work" is not a criterion.
3. **No suggested task breakdown** — Members need guidance. Include 2-4 suggested tasks with affected files.
4. **Forgetting to link documents** — Always link relevant docs to Epics so Members have context.
5. **Skipping human checkpoints** — Never auto-plan without human approval at key stages.
