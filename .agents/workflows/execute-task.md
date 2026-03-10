---
description: End-to-end workflow for executing a Story or Task from start to finish. Covers context gathering, blocker checks, planning, implementation, verification, commit, and summary with suggested next steps.
---

## Inputs
- **project_id** — The VibePM project ID or code
- **issue_id** — The VibePM issue ID or code (Story or Task)

---

## Step 1 — Gather Issue Context

**Goal:** Build a complete picture before planning anything.

1. \\\`projects(method: "get", projectId: "{project_id}")\\\` → get project statuses (note "In Progress", "In Review" IDs) and project code.

2. \\\`issues(method: "get", projectId: "{project_id}", issueId: "{issue_id}")\\\` → get issue details, parent issue (read its **full description**), sibling issues, all comments.

3. If parent is a **Story**, extract: acceptance criteria, task breakdown, technical notes, dependencies, out-of-scope items.

4. If the issue is a **Story with no child Tasks** → run the **Story Analysis Sub-flow** (see below), then restart this workflow on the first created Task.

5. \\\`notification(method: "list", isRead: false, limit: 10)\\\` → check for unaddressed feedback. Mark processed ones as read.

6. \\\`search(method: "query", projectId: "{project_id}", q: "<keywords>")\\\` → find related issues and documents.

---

## Step 2 — Check Blockers

**Goal:** If blocked, STOP immediately.

7. Check sibling issues for incomplete dependencies and blocking links.

8. Check comments for unresolved blockers or open questions.

**If blocked:**
- Post blocker comment: \\\`issues(method: "comment", ..., content: "⛔ Blocked by {dep} — {reason}")\\\`
- **🛑 STOP** — inform human with blocker details. Do NOT proceed.

**If clear:** Continue to Step 3.

---

## Step 3 — Gather Guidelines

**Goal:** Load team coding standards before planning.

9. \\\`docs(method: "list", projectId: "{project_id}")\\\` → scan for "Coding Guidelines", "PR Checklist", or similar docs.

10. Read each guidelines doc found. You MUST follow all rules.

11. If no guidelines doc exists → note it; skip guidelines compliance check later.

---

## Step 4 — Create Execution Plan

**Goal:** Present a reviewable plan BEFORE writing code.

12. Synthesize Steps 1–3 into:

\\\`\\\`\\\`markdown
## 🗺️ Execution Plan: {issueCode} — {title}

**What I'll implement:** <summary>
**Acceptance criteria:** <checklist>
**Technical approach:** <architecture, patterns, key decisions>
**Files to modify:** <list>
**Branch:** feat/{issue-code}-{slug}
**Dependencies:** {resolved / none}
**Risks:** <edge cases, concerns>
\\\`\\\`\\\`

13. Present plan and **⏸️ WAIT for approval**.

---

## Step 5 — Iterate Plan Until Approved

14. If human requests changes → update and re-present. Repeat until approved.

15. If human has questions → answer with references to acceptance criteria and context.

**Gate:** Do NOT proceed until human explicitly approves.

---

## Step 6 — Execute the Plan

16. Set status and assign: \\\`issues(method: "update", ..., statusName: "In Progress", assignToMe: true)\\\`

17. Create branch: \\\`feat/{issue-code}-short-slug\\\`

18. Implement following the approved plan. Make atomic commits: \\\`feat({issue-code}): description\\\`

19. If unexpected complexity/blocker/scope creep discovered → inform human, update plan, wait for re-approval.

---

## Step 7 — Verify Checklist

**Goal:** Loop until ALL items pass ✅.

20. **Acceptance Criteria** — verify each criterion from parent Story with evidence.

21. **Tests** — run test suite, fix failures related to your changes.

22. **Code Quality** — fix lint errors, resolve critical TODOs.

23. **Guidelines Compliance** (if doc exists) — check each guideline item.

24. **Git Conventions:**
    - Branch: \\\`feat/{issue-code}-*\\\`
    - All commits reference issue code: \\\`feat({issue-code}): ...\\\`
    - No untracked/unstaged changes

25. **Repeat 20–24** until every item ✅. Do NOT proceed with any ❌.

---

## Step 8 — Commit & Document

26. Ensure all changes committed with proper format.

27. Post implementation summary comment via \\\`issues(method: "comment", ...)\\\`:

\\\`\\\`\\\`markdown
## ✅ Implementation Complete

**Issue:** {issueCode} — {title}
**Branch:** feat/{issue-code}-{slug}

### Changes Made
- <files and modules changed, architecture decisions>

### Acceptance Criteria
- [x] <criterion> — {evidence}

### Tests
- **Status:** All passing / {N} pre-existing failures
- **New tests:** {list or "none"}

### Commits
- \\\\\\\`{hash}\\\\\\\` — {message}

### Known Limitations
- {edge cases not covered, or "None"}
\\\`\\\`\\\`

28. Update linked documents if needed: \\\`docs(method: "update", ...)\\\`

29. Move to "In Review": \\\`issues(method: "update", ..., statusName: "In Review")\\\`

---

## Step 9 — Summarize to Human

30. Present:

\\\`\\\`\\\`
✅ {issueCode} — {title} — COMPLETE.
🌿 Branch: feat/{issue-code}-{slug}
📋 Summary posted. Status: In Review.
Criteria: {N}/{N} met. Tests: passing. Guidelines: compliant.
\\\`\\\`\\\`

---

## Step 10 — Suggest Next Steps

31. Suggest:

\\\`\\\`\\\`
🔜 Next steps:
   → Reviewer: assess and approve/reject
   → If docs need updating: use **sync-docs** workflow
   → If sibling Tasks remain: run **execute-task** on the next one
   → If all Tasks Done: Story can be promoted to Done
\\\`\\\`\\\`

---

## Handling Review Feedback (Re-entry)

If the task is **rejected** or feedback is posted after Step 9:

32. Refresh comments via \\\`issues(method: "get", ...)\\\`.

33. Parse feedback into items:

    | # | Feedback | Category | Status |
    |---|---------|----------|--------|
    | 1 | <issue> | Must Fix | ⬜ |
    | 2 | <suggestion> | Should Fix | ⬜ |

    Categories: **Must Fix** (blocking), **Should Fix** (recommended), **Note** (informational).

34. If feedback unclear → post clarification, **🛑 STOP**, wait for response.

35. Address items: Must Fix → fix it. Should Fix → fix or note why deferred. Do NOT change anything beyond what was requested.

36. **Re-run Step 7** to verify nothing is broken.

37. Post rework comment:

\\\`\\\`\\\`markdown
## 🔄 Rework Complete — Round {N}

### Addressed Feedback
| # | Feedback | Resolution |
|---|---------|------------|
| 1 | <feedback> | ✅ <fix> |
| 2 | <feedback> | ⏭️ Deferred — <reason> |

### Tests: All passing
### Commits: \\\\\\\`{hash}\\\\\\\` — fix({code}): <msg>
\\\`\\\`\\\`

38. Move back to "In Review". Summarize to human.

---

## Story Analysis Sub-flow

When Step 1 item 4 triggers (Story with no child Tasks):

S1. Read project docs — \\\`docs(method: "list", ...)\\\` → read architecture, API, and data model docs. Read any docs linked to the Story or its Epic.

S2. Technical analysis — for each piece of work: affected files/modules, approach, risks, cross-cutting concerns (migrations, API changes, config, tests, security).

S3. Task sizing — 2–4 Tasks per Story. Each should be a meaningful unit completable in one session.
   ❌ Anti-patterns: one task per file, "write tests" as separate task, "research" as task, one giant task.

S4. Post implementation plan as comment on the Story. **⏸️ WAIT for human approval.**

S5. After approval, create Tasks: \\\`issues(method: "create", ..., type: "TASK", statusName: "Backlog")\\\`

S6. Restart this workflow on the first Task.

---

## Git Rules
- **Branch:** \\\`feat/{issue-code}-short-slug\\\`
- **Commits:** \\\`feat({issue-code}): description\\\` — ALL commits MUST reference issue code
- **Bug fixes:** \\\`fix({issue-code}): ...\\\` | **Refactors:** \\\`refactor({issue-code}): ...\\\`

## Constraints
- 🚫 NEVER code without human-approved plan
- 🚫 NEVER skip blocker checks
- 🚫 NEVER proceed past verify checklist with any ❌
- 🚫 NEVER promote to "Done" — only humans can
- 🚫 NEVER commit without issue code reference
- 🚫 NEVER ignore review feedback
- ✅ ALWAYS read parent Story acceptance criteria before planning
- ✅ ALWAYS load guidelines before implementation
- ✅ ALWAYS wait for plan approval before coding
- ✅ ALWAYS verify all checklist items before committing`,
};
