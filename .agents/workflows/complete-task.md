---
description: Post-implementation checklist: verify acceptance criteria, run tests, document changes, hand off for review.
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
# Workflow: Complete Task

Post-implementation checklist: verify all acceptance criteria are met, run tests,
document changes with a structured comment, handle partial completion, and hand off for review.

## Inputs
- **project_id** — The VibePM project ID or code
- **issue_id** — The VibePM issue ID or code that was implemented

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Self-Review

1. Call `issues(method: "get", projectId: "{project_id}", issueId: "{issue_id}")` to refresh:
   - The original requirements and acceptance criteria
   - Any comments added during implementation
   - Parent Story description for acceptance criteria reference

2. **Check acceptance criteria** from the parent Story:

   For each criterion, verify:
   | Criterion | Met? | Evidence |
   |-----------|------|----------|
   | <criterion 1> | ✅/❌ | <how you verified> |

   - If ANY criterion is NOT met:
     a. Can you fix it now? → Fix it, then re-run this checklist.
     b. Is it out of scope or blocked? → Note it in the comment and proceed.

## Phase 2 — Verify Quality

3. Run project tests:
   - Execute the project's test suite
   - Record: pass count, fail count, coverage (if available)
   - If tests fail: fix if related to your changes, note if pre-existing.

4. Check code quality:
   - Are there any lint errors?
   - Any TODO comments that should be resolved?

5. **Team Guidelines Check:**
   - Call \`docs(method: "list", projectId: "{project_id}")\` — scan briefs for a doc titled "Coding Guidelines" or similar in the Guidelines folder
   - Call \`docs(method: "get", projectId: "{project_id}", documentId: "<guidelines-doc-id>")\` — read the full guidelines
   - Self-review your code changes against EACH item in the team's checklist
   - Record compliance:

     | Guideline | Status | Notes |
     |-----------|--------|-------|
     | <item 1> | ✅/❌ | <detail> |

   - If any item fails: fix it before proceeding
   - If no Guidelines doc exists: note it in the comment ("No team guidelines doc found — skipped guidelines check")

6. Verify all commits reference the issue code:
   - Pattern: `feat({issue-code}): description`

## Phase 3 — Document Changes

7. Call `issues(method: "comment", projectId: "{project_id}", issueId: "{issue_id}")` with:

   ```markdown
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
   - \`{hash}\` — {message}

   ### Known Limitations
   - {Any edge cases not covered}

   ### Documentation Impact
   - {List linked/updated documents, or "None"}
   ```

## Phase 4 — Transition

8. Check if any documents linked to this issue need updating. If so, update them using `docs(method: "update", ...)`.

9. Determine the right next status:
   - **All criteria met → Move to "In Review"**
   - **Partially complete but usable → Move to "In Review" with notes**
   - **Blocked or fundamentally incomplete → Stay "In Progress"**

10. If other docs need updating, use the **sync-docs** workflow.

11. Confirm to the human:
    ```
    ✅ Task {issueCode} moved to In Review.
    📋 Implementation summary posted as comment.
    📄 Documentation impact: {needs sync / none}.
    ⏭️ A reviewer will now assess the implementation.
    ```

## Git Verification

Before moving to In Review, verify:

- **Branch name** follows convention: \`feat/{issue-code}-*\`
- **All commits** on this branch reference the issue code
- **Commit format** is correct: \`feat({issue-code}): description\`
- No untracked or unstaged changes remain

## Constraints
- 🚫 NEVER move an issue to "Done" — only humans can approve
- 🚫 NEVER skip the acceptance criteria check
- 🚫 NEVER move to In Review with uncommitted changes
- ✅ ALWAYS check team guidelines doc (if it exists) before moving to In Review
- ✅ ALWAYS post an implementation summary comment before moving to In Review
- ✅ ALWAYS include test results in the summary
- ✅ ALWAYS verify git commit conventions before completing

## Summary & Next Steps

When this workflow completes, present:

\`\`\`
✅ Task {issueCode} moved to In Review.
📋 Implementation summary posted as comment.
🌿 Branch: feat/{issue-code}-{slug}
📄 Documentation impact: {needs sync / none}.

🔜 Suggested next steps:
   → A reviewer will use the **review-task** workflow to assess the implementation
   → If documentation needs updating, use the **sync-docs** workflow
   → If sibling Tasks remain under the Story, use **start-task** on the next one
\`\`\`
---
_Integrity: 147 lines · workflow:complete-task · DO NOT MODIFY_
