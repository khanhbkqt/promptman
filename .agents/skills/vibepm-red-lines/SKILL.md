---
name: vibepm-red-lines
description: |
  Backend-enforced rules that agents CANNOT bypass. Understand these before
  you hit a 403 error. Covers status transition guards, actor type detection,
  and StrictResourceGuard error handling.
  Use when "403 error", "permission denied", "cannot move issue", "red lines",
  "what can't I do", or "backend rules".
---

# VibePM Red Lines & Error Handling

These rules are enforced by the backend. You cannot bypass them.

## Hard Rules (403 Forbidden)

### Rule 1: Agents Cannot Move Issues Out of "In Review"

Only humans (authenticated via web session) can transition issues **from** "In Review"
to any other status (including "Done").

**Why:** The "In Review" → "Done" gate is the final human approval checkpoint.
Even if you review and approve work, a human must sign off.

**What you'll see if you try:**
```json
{
  "statusCode": 403,
  "message": "Only human users can transition issues from 'In Review'. Agent actions are restricted at this stage."
}
```

**What to do instead:** Post your review comment, then tell the human: "This issue is ready to move to Done."

### Rule 2: Agents Cannot Promote Epics/Stories Off Backlog

Only humans can move an **Epic** or **Story** from "Backlog" to any other status.

**Why:** Promoting planning items out of backlog means committing to work — that decision
belongs to humans.

**What you'll see if you try:**
```json
{
  "statusCode": 403,
  "message": "Only human users can move Epics and Stories out of Backlog."
}
```

**What to do instead:** Create Epics/Stories in Backlog status. Tell the human when
items are ready to be promoted.

**Note:** Tasks and Bugs can be freely moved by agents (except out of "In Review").

## Actor Type Detection

The backend classifies every request:

| Auth Method | Actor Type | Capabilities |
|------------|-----------|--------------|
| API Key (`Authorization: Bearer vpm_...`) | `agent` | Cannot move from "In Review", cannot promote Epic/Story off Backlog |
| JWT / Session Cookie | `human` | Full access |

Every mutation you make is recorded in the activity log with `actorType: "agent"`,
so humans can always distinguish AI-originated changes from human ones.

## StrictResourceGuard Errors

When you pass an invalid ID (project, issue, status, etc.), the backend returns
the valid options in the error body. **Read the error carefully — it tells you
what IDs are valid.**

**Example: Invalid project ID**
```json
{
  "statusCode": 400,
  "message": "Invalid projectId \"nonexistent\". Valid projects: THN (id: abc-123), PRJ (id: def-456)"
}
```

**Example: Invalid status ID**
```json
{
  "statusCode": 400,
  "message": "Invalid statusId \"wrong-id\". Valid statuses for project THN: Backlog (id: s1), In Progress (id: s2), In Review (id: s3), Done (id: s4)"
}
```

**Best practice:** Always call `projects(method: "get")` first to retrieve valid status IDs
before creating or updating issues.

## Summary: What You CAN and CANNOT Do

| Action | Agent | Human |
|--------|-------|-------|
| Create issues (any type) | ✅ | ✅ |
| Move Task/Bug between statuses | ✅ (except from "In Review") | ✅ |
| Move Epic/Story off Backlog | ❌ | ✅ |
| Move any issue from "In Review" | ❌ | ✅ |
| Add comments | ✅ | ✅ |
| Create/update documents | ✅ | ✅ |
| Link documents to issues | ✅ | ✅ |
| Create planning boards | ✅ | ✅ |
