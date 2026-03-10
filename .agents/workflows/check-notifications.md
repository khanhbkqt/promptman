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
# Workflow: Check Notifications

Triage unread notifications before starting new work. Ensures you don't miss
review feedback, new assignments, or status changes on tasks you care about.

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — List Unread

1. Call `notification(method: "list", isRead: false, limit: 20)`
2. If zero unread → "No unread notifications. You're all caught up." → STOP

## Phase 2 — Triage by Type

For each notification, act based on type:

- **COMMENTED** (highest priority) → Read the comment thread. If it's review
  feedback on your work, address it before starting new tasks.
  `issues(method: "get", projectId: "…", issueId: "…")`

- **ASSIGNED** → New work assigned to you. Call `issues(method: "get", ...)` to
  understand the assignment. Decide: pick up now or defer based on priority.

- **STATUS_CHANGED** → A task you care about moved. If a blocker was resolved,
  check if you can now proceed on blocked work.

- **MENTIONED** → Someone referenced you. Read the context and respond if needed.

- **WATCHER_UPDATED** → You were added/removed as a watcher. Note for awareness.

## Phase 3 — Mark as Read

After processing all notifications:

```
notification(method: "read_all")
```

Or mark individual ones:
```
notification(method: "read", notificationId: "<id>")
```

## Constraints
- 🔴 Process COMMENTED notifications first — review feedback is highest priority
- 🔴 Don't start new work until review feedback is addressed
- ✅ Keep triage brief — if action is needed, note it and move on
- ✅ Use `issues(method: "get", ...)` to understand assignments before deciding
---
_Integrity: 55 lines · workflow:check-notifications · DO NOT MODIFY_