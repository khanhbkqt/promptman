---
description: End-to-end workflow for executing a Story or Task from start to finish. Covers context gathering, blocker checks, planning, implementation, verification, commit, and summary with suggested next steps.
---

## Inputs
- **project_id** — The VibePM project ID or code
- **issue_id** — The VibePM issue ID or code (Story or Task)

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- \\\`<toolname>(method: "help")\\\` — list all available commands
- \\\`<toolname>(method: "help", command: "<name>")\\\` — show parameters for a specific command

This works for all tools: \\\`issues\\\`, \\\`docs\\\`, \\\`projects\\\`, \\\`search\\\`, \\\`reports\\\`, \\\`notification\\\`.

---

## Step 1 — Gather Issue Context

**Goal:** Build a complete picture of the task before planning anything.

1. Call \\\`projects(method: "get", projectId: "{project_id}")\\\` to retrieve:
   - Project statuses → note the "In Progress" and "In Review" status IDs
   - Project code → for branch naming

2. Call \\\`issues(method: "get", projectId: "{project_id}", issueId: "{issue_id}")\\\` to retrieve:
   - Issue details: title, type, description, priority, current status
   - Parent issue (Story or Epic) — read its **full description**
   - Sibling issues (other Tasks under the same parent)
   - All existing comments — scan for unaddressed feedback or blockers

3. If the parent is a **Story**, extract from its description:
   - **Acceptance criteria** → these define "done"
   - **Suggested task breakdown** → confirm your task matches
   - **Technical notes** → implementation guidance
   - **Dependencies** → are prerequisite Stories/Tasks complete?
   - **Out of scope** → what NOT to do

4. If the issue itself is a **Story with no child Tasks**:
   - Read the Story's suggested task breakdown
   - Create Tasks under this Story using \\\`issues(method: "create", ...)\\\`
   - Then restart this workflow on the first Task

5. Check notifications for relevant updates:
   - Call \\\`notification(method: "list", isRead: false, limit: 10)\\\`
   - Look for comments, status changes, or review feedback on this task
   - Mark processed: \\\`notification(method: "read", notificationId: "<id>")\\\`

6. Search for related issues and documents:
   - Call \\\`search(method: "query", projectId: "{project_id}", q: "<relevant keywords>")\\\`
   - Check if linked documents exist for the issue

**Output of this step:** A mental model of what needs to be built, the acceptance criteria, and all relevant context.

---

## Step 2 — Check Blockers

**Goal:** Confirm the task can proceed. If blocked, STOP immediately.

7. Review sibling issues (from Step 1):
   - Are there Tasks that yours depends on that are still incomplete?
   - Are there blocking links on the issue?

8. Review comments for unresolved blockers or open questions.

9. Call \\\`issues(method: "list", projectId: "{project_id}", mine: true)\\\` to check your current workload.

**If blocked:**
   a. Post a comment on the issue explaining the blocker:
      \\\`issues(method: "comment", projectId: "{project_id}", issueId: "{issue_id}", content: "⛔ Blocked by {dependency} — {reason}")\\\`
   b. **🛑 STOP — Inform the human:**
      \\\`\\\`\\\`
      ⛔ Task {issueCode} is BLOCKED.
      Reason: {dependency} is not yet complete.
      Action needed: {what needs to happen before this can proceed}
      \\\`\\\`\\\`
   c. Do NOT proceed to Step 3. Wait for human direction.

**If not blocked:** Continue to Step 3.

---

## Step 3 — Gather Guidelines Documents

**Goal:** Load team coding standards so the execution plan and implementation align with project conventions.

10. Call \\\`docs(method: "list", projectId: "{project_id}")\\\` and scan for:
    - Documents titled "Coding Guidelines", "PR Checklist", or similar
    - Documents in a "Guidelines" folder or category

11. For each guidelines doc found:
    - Call \\\`docs(method: "get", projectId: "{project_id}", documentId: "<doc-id>")\\\`
    - Read and memorize every rule — you MUST follow all of them

12. If **no guidelines doc exists**: Note it — you'll skip the guidelines compliance check later.

**Output of this step:** A list of team rules and conventions to follow during implementation.

---

## Step 4 — Create Execution Plan

**Goal:** Present a clear, reviewable plan to the human BEFORE writing any code.

13. Synthesize everything from Steps 1–3 into a structured execution plan:

\\\`\\\`\\\`markdown
## 🗺️ Execution Plan: {issueCode} — {title}

**What I'll implement:** <1-2 sentence summary>

**Acceptance criteria I'll satisfy:**
- [ ] <criterion 1>
- [ ] <criterion 2>

**Technical approach:**
- <How you plan to implement — architecture, patterns, key decisions>
- <Why this approach was chosen over alternatives>

**Files I expect to modify:**
- <file/module list with brief description of changes>

**Branch:** feat/{issue-code}-{slug}
**Dependencies:** {resolved / none}
**Guidelines compliance:** {list key guidelines that apply}
**Estimated scope:** {small / medium / large}

**Risks & edge cases:**
- <anything that could go wrong or needs special attention>
\\\`\\\`\\\`

14. Present this plan to the human and **⏸️ WAIT for approval**.

---

## Step 5 — Iterate Plan Until Approved

**Goal:** Incorporate human feedback until the plan is approved.

15. If the human requests changes:
    a. Update the plan based on feedback
    b. Re-present the updated plan
    c. Repeat until the human explicitly approves

16. If the human has questions:
    a. Answer them with references to the acceptance criteria and technical context
    b. Do NOT proceed to implementation until questions are resolved

**Gate:** Do NOT proceed to Step 6 until the human says "approved", "looks good", "go ahead", or similar affirmative.

---

## Step 6 — Execute the Plan

**Goal:** Implement the approved plan.

17. Set status and assign:
    - Call \\\`issues(method: "update", projectId: "{project_id}", issueId: "{issue_id}", statusName: "In Progress", assignToMe: true)\\\`

18. Create a feature branch:
    - Naming: \\\`feat/{issue-code}-short-slug\\\` (e.g. \\\`feat/PRJ-42-search-api\\\`)

19. Implement the changes following the approved plan:
    - Follow the team guidelines (from Step 3)
    - Reference acceptance criteria continuously
    - Make atomic commits with correct format: \\\`feat({issue-code}): description\\\`

20. During implementation, if you discover:
    - **Unexpected complexity** → Inform human, update the plan, wait for re-approval
    - **A blocker** → STOP, post blocker comment, inform human (same as Step 2)
    - **Scope creep** → Check "Out of scope" from the Story; ask human if unsure

---

## Step 7 — Verify Checklist

**Goal:** Confirm everything meets quality standards. Loop until ALL items pass.

21. **Acceptance Criteria Check:**

    For each criterion from the parent Story:
    | Criterion | Met? | Evidence |
    |-----------|------|----------|
    | <criterion 1> | ✅/❌ | <how you verified> |

    - If ANY criterion is NOT met:
      a. Can you fix it now? → Fix it, re-verify.
      b. Out of scope or blocked? → Note it for the summary.

22. **Run Tests:**
    - Execute the project's test suite
    - Record: pass count, fail count, coverage
    - If tests fail: fix if related to your changes; note if pre-existing

23. **Code Quality:**
    - Lint errors? Fix them.
    - TODO comments for critical functionality? Resolve them.

24. **Guidelines Compliance** (if guidelines doc exists):

    | Guideline | Status | Notes |
    |-----------|--------|-------|
    | <item 1> | ✅/❌ | <detail> |

    - If any item fails → fix it, then re-run this checklist.

25. **Git Convention Verification:**
    - Branch name follows: \\\`feat/{issue-code}-*\\\`
    - ALL commits reference the issue code: \\\`feat({issue-code}): description\\\`
    - No untracked or unstaged changes remain
    - Use \\\`fix({issue-code}): ...\\\` for bug fixes within the task
    - Use \\\`refactor({issue-code}): ...\\\` for refactoring within the task

26. **Repeat Steps 21–25** until every item shows ✅. Do NOT proceed with any ❌.

---

## Step 8 — Commit & Document

**Goal:** Finalize the code and post a structured implementation summary.

27. Ensure all changes are committed with proper format:
    - \\\`feat({issue-code}): <description>\\\`
    - Every commit MUST reference the issue code — no exceptions

28. Post an implementation summary comment:

    Call \\\`issues(method: "comment", projectId: "{project_id}", issueId: "{issue_id}", content: "...")\\\` with:

    \\\`\\\`\\\`markdown
    ## ✅ Implementation Complete

    **Issue:** {issueCode} — {title}
    **Branch:** feat/{issue-code}-{slug}

    ### Changes Made
    - <What was implemented — be specific about files and modules>
    - <Architecture decisions made and why>

    ### Acceptance Criteria
    - [x] <criterion 1> — {brief evidence}
    - [x] <criterion 2> — {brief evidence}

    ### Tests
    - **Status:** All passing / {N} failures (pre-existing)
    - **New tests added:** {list or "none"}

    ### Commits
    - \\\\\\\`{hash}\\\\\\\` — {message}

    ### Known Limitations
    - {Any edge cases not covered, or "None"}

    ### Documentation Impact
    - {List linked/updated documents, or "None"}
    \\\`\\\`\\\`

29. Check if linked documents need updating:
    - If yes → update them using \\\`docs(method: "update", ...)\\\`

30. Move status to "In Review":
    - Call \\\`issues(method: "update", projectId: "{project_id}", issueId: "{issue_id}", statusName: "In Review")\\\`

---

## Step 9 — Summarize to Human

**Goal:** Present a clear completion summary.

31. Present the following summary:

\\\`\\\`\\\`
✅ Task {issueCode} — {title} — COMPLETE.
🌿 Branch: feat/{issue-code}-{slug}
📋 Implementation summary posted as comment.
📄 Documentation impact: {updated X / none}.
🔍 Status: Moved to In Review.

Acceptance criteria: {N}/{N} met.
Tests: All passing.
Guidelines: All compliant.
\\\`\\\`\\\`

---

## Step 10 — Suggest Next Steps

**Goal:** Guide the human on what to do next.

32. Based on the project state, suggest relevant next actions:

\\\`\\\`\\\`
🔜 Suggested next steps:
   → Human reviewer: assess the implementation and approve/reject
   → If documentation needs updating: use the **sync-docs** workflow
   → If sibling Tasks remain under the Story: run **execute-task** on the next one
   → If all Tasks under the Story are Done: the Story can be promoted to Done
\\\`\\\`\\\`

---

## Handling Review Feedback (Re-entry)

If the human or a reviewer **rejects** the task or leaves feedback comments
after Step 9, re-enter this workflow at this section:

33. Call \\\`issues(method: "get", ...)\\\` to refresh comments and status.

34. Find the **most recent review comment** (look for "## Review" or feedback markers).

35. Parse feedback into actionable items:

    | # | Feedback Item | Category | Status |
    |---|--------------|----------|--------|
    | 1 | <specific issue> | Must Fix | ⬜ Pending |
    | 2 | <specific issue> | Should Fix | ⬜ Pending |

    **Categories:**
    - **Must Fix** — blocking approval, MUST be addressed
    - **Should Fix** — recommended, not blocking
    - **Note** — informational, no action needed

36. If any feedback is unclear or you disagree:
    - Post a clarification comment
    - **🛑 STOP** and wait for response

37. Address each item:
    - **Must Fix** → make the change, mark as addressed
    - **Should Fix** → fix if reasonable; if not, note why in the rework comment
    - Do NOT refactor or change anything beyond what was requested

38. **Re-run Step 7 (Verify Checklist)** — ensure nothing is broken.

39. Post a rework summary comment:

    \\\`\\\`\\\`markdown
    ## 🔄 Rework Complete

    **Issue:** {issueCode} — {title}
    **Review round:** {N}

    ### Addressed Feedback
    | # | Feedback | Resolution |
    |---|---------|------------|
    | 1 | <original feedback> | ✅ <what was changed> |
    