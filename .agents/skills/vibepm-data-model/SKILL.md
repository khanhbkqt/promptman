---
name: vibepm-data-model
description: |
  Understand VibePM's core data model — the issue hierarchy, status workflow,
  project codes, and how to resolve IDs. Read this FIRST before using any tools.
  Use when "what is VibePM", "data model", "issue types", "status workflow",
  "how do statuses work", "what is an Epic", or "explain VibePM structure".
---

# VibePM Data Model

This skill teaches you how VibePM organizes work. Read this before using any tools.

## Issue Hierarchy

VibePM uses a strict parent-child hierarchy:

```
Epic (Feature)
├── Story (Shippable Deliverable)
│   ├── Task (Implementation Step)
│   ├── Task
│   └── Bug (Defect Fix)
└── Story
    ├── Task
    └── Task
```

| Type | Created by | Purpose |
|------|-----------|---------|
| **Epic** | PM agent | A feature or initiative. Groups related Stories. |
| **Story** | PM agent | An independently shippable deliverable. Contains acceptance criteria and suggested task breakdown. |
| **Task** | Member agent | A technical implementation step. Always under a Story. |
| **Bug** | Any agent | A defect to fix. Can be standalone or under a Story. |

**Key rule:** PM creates Epics and Stories. Members create Tasks under Stories.

## Status Workflow

Every project defines its own statuses. A typical flow:

```
Backlog → In Progress → In Review → Done
```

- **Backlog** — Not started. New issues land here.
- **In Progress** — A human or agent is actively working on it.
- **In Review** — Implementation complete, awaiting reviewer/PM feedback.
- **Done** — Approved and completed. **Only humans can move issues here.**

### How to get status IDs

Statuses are project-specific. You need the UUID to set/change status:

```
projects(method: "get", projectId: "PRJ")
→ returns { statuses: [{ id: "uuid-1", name: "Backlog" }, { id: "uuid-2", name: "In Progress" }, ...] }
```

Then use the `id` value when calling `issues(method: "create", statusId: "uuid-1")` or `issues(method: "update", statusId: "uuid-2")`.

## Identifiers

### Project Codes

Every project has a short code (e.g., `THN`, `PRJ`). Most tools accept **either**:
- Project UUID: `"a1b2c3d4-..."`
- Project code: `"THN"`

Both work interchangeably for the `projectId` parameter in all tools.

### Issue Codes

Issues get auto-generated codes based on the project: `{PROJECT_CODE}-{NUMBER}`.

Example: `THN-42` means the 42nd issue in the THN project.

Most tools accept **either**:
- Issue UUID: `"e5f6g7h8-..."`
- Issue code: `"THN-42"`

Both work interchangeably for the `issueId` parameter.

### Document IDs

Documents have:
- UUID: `"d1e2f3g4-..."`
- Slug: `"architecture-overview"` (auto-generated from title)

The `docs(method: "get")` tool accepts either UUID or slug.

## Planning Boards

Planning Boards are cross-project sprint boards that group Stories for a sprint or phase:

```
Planning Board: "Sprint 5"
├── Story THN-10 (from project THN)
├── Story PRJ-25 (from project PRJ)
└── Story THN-15 (from project THN)
```

- **KANBAN** boards use a shared workflow (status columns)
- **LIST** boards are simple aggregations
- Use `manage_board_stories` to add/remove Stories from a board

## Documents as Knowledge Base

VibePM has a hierarchical document system per project:

```
📄 Architecture Overview (brief: "System architecture, tech stack, and deployment")
├── 📄 API Reference (brief: "REST endpoints and request/response schemas")
└── 📄 Data Model (brief: "Database schema and entity relationships")
📄 Getting Started (brief: "Setup guide for new developers")
```

- The `brief` field is a 1-2 sentence summary — use it to decide whether to read the full document
- Documents can be linked to issues for context
- Always call `docs(method: "list")` first and scan `brief` fields before fetching full content

## Notifications

Notifications are created automatically when events happen in VibePM:

| Type | Trigger |
|------|---------|
| `ASSIGNED` | An issue was assigned to you |
| `COMMENTED` | Someone commented on an issue you watch |
| `STATUS_CHANGED` | An issue you watch changed status |
| `MENTIONED` | Someone @mentioned you |
| `WATCHER_UPDATED` | You were added/removed as a watcher |

Key fields agents care about: `type`, `title`, `message`, `isRead`, `entityType`, `entityId`.

Access via the `notification` tool with method dispatch:

```
notification(method: "list", isRead: false)     → unread notifications
notification(method: "read", notificationId: "…") → mark one as read
notification(method: "read_all")                → mark all as read
notification(method: "unread_count")            → get unread count
```

## Quick Reference: Discovery Flow

When you first connect, follow this sequence to orient yourself:

```
0. issues(method: "help")                          → discover available commands & params
1. projects(method: "list")                        → see all projects
2. projects(method: "get", projectId: "PRJ")       → get statuses, project details  
3. docs(method: "list", projectId: "PRJ")          → scan knowledge base (read briefs)
4. issues(method: "list", projectId: "PRJ", mine: true)  → see your assigned work
5. issues(method: "get", projectId: "PRJ", issueId: "PRJ-42") → deep-dive one issue
6. notification(method: "list", isRead: false)  → check unread notifications
```

> **💡 Tip:** Unsure about a tool's parameters? Call it with `method: "help"` to list commands,
> or `method: "help", command: "<name>"` for parameter details.
