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
# Workflow: Rework Task

Handle a review rejection: read the reviewer feedback, parse required changes,
   address each item, re - verify, and resubmit for review.

## Inputs
      - ** project_id ** — The VibePM project ID or code
         - ** issue_id ** — The rejected issue ID to rework

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Understand Feedback

1. Call `issues(method: "get", projectId: "{project_id}", issueId: "{issue_id}")` to retrieve:
   - All comments — find the **most recent review comment** (look for "## Review" or "## 🔄 Review")
   - Current status — should be "In Progress" (sent back by reviewer)
   - Parent Story — refresh on acceptance criteria

2. Parse the review comment into actionable items:

   | # | Feedback Item | Category | Status |
   |---|--------------|----------|--------|
   | 1 | <specific issue> | Must Fix | ⬜ Pending |
   | 2 | <specific issue> | Must Fix | ⬜ Pending |
   | 3 | <suggestion> | Should Fix | ⬜ Pending |

   **Categories:**
   - **Must Fix** — blocking approval, MUST be addressed
   - **Should Fix** — recommended but not blocking
   - **Note** — informational, no action needed

3. If any feedback is unclear or you disagree:
   - Post a comment asking for clarification
   - ⚠️ **STOP** and wait for response before proceeding

## Phase 2 — Address Feedback

4. For each "Must Fix" item:
   a. Understand exactly what needs to change
   b. Make the code change
   c. Mark as addressed

5. For each "Should Fix" item:
   a. Evaluate if reasonable to address now
   b. If yes, fix it. If no, note why in the rework comment.

6. **Do NOT refactor or change anything beyond what was requested.**

## Phase 3 — Re-verify

7. Run the same verification as the complete-task workflow:
   - Re-check acceptance criteria from the parent Story
   - Run tests — ensure nothing is broken

8. Check if any new commits need issue code references:
   - Pattern: `fix({issue-code}): address review feedback`

## Phase 4 — Resubmit

9. Post a rework summary comment using `issues(method: "comment", ...)`:

   ```markdown
   ## 🔄 Rework Complete

   **Issue:** {issueCode} — {title}
   **Review round:** {N}

   ### Addressed Feedback
   | # | Feedback | Resolution |
   |---|---------|------------|
   | 1 | <original feedback> | ✅ <what was changed> |
   | 2 | <original feedback> | ✅ <what was changed> |
   | 3 | <should fix item> | ⏭️ Deferred — <reason> |

   ### Changes Made
   - <specific file/module changes>

   ### Tests
   - **Status:** All passing

   ### Commits
   - \`{hash}\` — fix({code}): <message>
   ```

10. Move the issue back to "In Review" using `issues(method: "update", ...)`.

11. Confirm:
    ```
    🔄 Rework complete for {issueCode}.
    ✅ {N} must-fix items addressed.
    📋 Rework summary posted as comment.
    ⏭️ Moved back to In Review for re-assessment.
    ```

## Constraints
- 🚫 NEVER ignore "Must Fix" items — they are blocking
- 🚫 NEVER rewrite code beyond what was requested
- 🚫 NEVER move to In Review without posting a rework comment
- ✅ ALWAYS reference the specific feedback items in your rework comment
- ✅ ALWAYS run tests after making fixes
---
_Integrity: 107 lines · workflow:rework-task · DO NOT MODIFY_