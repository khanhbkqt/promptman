---
name: vibepm-review
description: |
  Review completed work in VibePM — gather context, assess implementation
  against acceptance criteria, post structured feedback, and manage the
  approval/rejection workflow.
  Use when "review task", "code review", "assess implementation", "approve",
  "reject", "changes requested", or "review workflow".
---

# VibePM Review Workflow

This skill covers how to review completed work: gathering context, assessing quality,
posting structured feedback, and managing the approval/rejection flow.

## The Review Flow

```
Issue in "In Review" → Gather Context → Assess → Post Feedback → Approve or Reject
```

**Key constraint:** Even if you approve, you CANNOT move issues to "Done".
Only humans can finalize. You can only move issues BACK to "In Progress" for rework.

## Step 1: Gather Context

```
issues(method: "get", projectId: "PRJ", issueId: "PRJ-52")
→ Returns:
  - The Task description and requirements
  - Parent Story with acceptance criteria (your review checklist)
  - Implementation summary comment (posted by the developer)
  - All prior comments (clarifications, decisions)
  - Linked documents (technical context)
```

**Critical: Read the implementation summary comment.** It should contain:
- Branch name (so you can find the code)
- Changes made
- Acceptance criteria status
- Test results
- Known limitations

**Red flag:** If there's no implementation summary, note this in your review.

## Step 2: Assess the Implementation

Check against the parent Story's acceptance criteria:

| Criterion | Status | Notes |
|-----------|--------|-------|
| Returns search results | ✅ Verified | Endpoint works with sample data |
| Supports pagination | ✅ Verified | limit/offset params functional |
| Response time < 200ms | ⚠️ Not verified | No performance test mentioned |

Also check:
- **Tests:** Are they mentioned? Do they cover edge cases?
- **Code quality:** Follows project patterns? Any security concerns?
- **Documentation impact:** Were affected docs updated?
- **Commit messages:** Do they reference the issue code?

## Step 3: Post Structured Feedback

Always use this format:

```
issues(method: "comment", projectId: "PRJ", issueId: "PRJ-52",
  content: "## Review

**Decision:** Approved ✅ / Changes Requested 🔄

**Findings:**
- <finding 1 — be specific about files/patterns>
- <finding 2>

**Action Required:**
- <None / specific changes with clear instructions>")
```

## Step 4a: Approve

When everything looks good:

```
1. Post the approval comment (Step 3 with "Approved ✅")

2. Inform the human:
   "PRJ-52 has been reviewed and approved. A human should move it to Done."

3. DO NOT call issues(method: "update") to change status.
   → The backend will return 403 if you try to move from "In Review"
```

## Step 4b: Reject (Changes Requested)

When changes are needed:

```
1. Post the review comment with specific, actionable feedback:
   issues(method: "comment", projectId: "PRJ", issueId: "PRJ-52",
     content: "## Review\n\n**Decision:** Changes Requested 🔄\n\n**Findings:**\n- Search parameter passed directly to SQL — injection risk\n- No input length validation\n\n**Action Required:**\n1. Use parameterized queries or ORM methods\n2. Add max length validation (suggest 500 chars)\n3. Add test for SQL injection attempt")

2. Move back to In Progress:
   issues(method: "update", projectId: "PRJ", issueId: "PRJ-52",
     statusId: "<in-progress-status-id>")

3. The developer will see the feedback and rework.
```

## Checking for Review Patterns

Scan the comment history in `issues(method: "get")` to identify issues that have been bounced multiple times:

→ Look for repeated "Changes Requested" feedback or "In Review" cycles in the comment log.

If bounced 3+ times:
→ Flag as a process concern in your review comment
→ Suggest the developer and PM discuss scope or approach
```

## Example: Full Review — Approved

```
1. issues(method: "get", projectId: "PRJ", issueId: "PRJ-52")
   → Task: "Search query builder"
   → Parent Story acceptance criteria:
     - [ ] Returns matching results
     - [ ] Supports pagination
     - [ ] < 200ms response time
   → Implementation comment by developer:
     "Branch: feat/PRJ-52-search-query-builder
      Added QueryBuilder, 12 new tests, all passing"

2. Review code on the branch
   → Parameterized queries ✅, pagination util used ✅, tests cover edge cases ✅

3. issues(method: "comment", projectId: "PRJ", issueId: "PRJ-52",
     content: "## Review\n\n**Decision:** Approved ✅\n\n**Findings:**\n- QueryBuilder correctly uses parameterized queries\n- Pagination follows existing project patterns\n- Tests cover happy path, empty results, and max limit\n\n**Action Required:** None. Ready for human to promote to Done.")

4. Tell the human: "PRJ-52 reviewed and approved. Ready for Done."
```

## Example: Full Review — Changes Requested

```
1. issues(method: "get", projectId: "PRJ", issueId: "PRJ-55")
   → Implementation comment shows no mention of input validation

2. issues(method: "comment", projectId: "PRJ", issueId: "PRJ-55",
     content: "## Review\n\n**Decision:** Changes Requested 🔄\n\n**Findings:**\n- Search parameter `q` is interpolated directly into SQL string (line 42 of search.service.ts)\n- No length validation — DoS risk with extremely long queries\n\n**Action Required:**\n1. Replace string interpolation with parameterized query\n2. Add Zod validation: max 500 characters\n3. Add test case for injection attempt")

3. issues(method: "update", projectId: "PRJ", issueId: "PRJ-55",
     statusId: "<in-progress-status-id>")
```

## Review Best Practices

- **Be specific** — Reference file names and line numbers, not "fix the security issue"
- **Distinguish severity** — "Must Fix" vs "Should Fix" vs "Nice to Have"
- **Check the summary comment** — If missing, that's a review finding itself
- **Check test results** — No tests mentioned = red flag
- **Check doc impact** — Did the developer update affected docs?
