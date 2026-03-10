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
# Workflow: Review Task

Conduct a structured review of completed work: gather context, examine implementation
against requirements, check for human reviewer comments, provide detailed feedback,
and make an approve / reject decision. Also covers responding to follow-up comments.

## Inputs
   - **project_id** — The VibePM project ID or code
   - **issue_id** — The VibePM issue ID or code to review

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Gather Full Context

1. Call `issues(method: "get", projectId: "{project_id}", issueId: "{issue_id}")` to retrieve:
   - Issue details (title, type, description, status)
   - Parent Story (for acceptance criteria reference)
   - Child issues (if reviewing a Story, check all Tasks)
   - Comments (implementation notes from the Member)

2. If the issue has a parent Story, call `issues(method: "get", ...)` on the parent too to read the full acceptance criteria.

3. Call `issues(method: "list", projectId: "{project_id}", parentId: "{issue_id}")` if this is a Story to check all child Tasks.

## Phase 2 — Check for Human Comments

4. Scan all comments for **unacknowledged human feedback** — comments posted
   AFTER the implementation summary that are NOT from the agent. For each:

   **Classify the comment:**

   | Type | What it looks like | Action |
   |------|--------------------|--------|
   | **Question** | "Why did you use X?", "What about Y?" | Post a detailed answer referencing the code |
   | **Minor Fix Request** | "Can you rename this?", "Add a null check" | Make the fix, run tests, post update comment |
   | **Discussion** | Thinking aloud, exploring trade-offs | Respond thoughtfully, then **STOP** and wait |
   | **Approval with Notes** | "Looks good, but consider X later" | Acknowledge the notes, proceed with approval |

5. **If there are unaddressed comments:**
   a. Respond to each comment via `issues(method: "comment", ...)`
   b. If a minor fix was requested: make the change, run tests, then post:
      ```markdown
      ## 📝 Addressed Feedback

      **Re:** {summary of the comment}
      **Action:** {what was changed}
      **Tests:** Still passing
      ```
   c. If the comment is a question or discussion: respond and **STOP** — do not
      proceed to the decision phase until the reviewer signals readiness.

## Phase 3 — Verify Team Guidelines & Conventions

6. Call \`docs(method: "list", projectId: "{project_id}")\` and scan for team documents like "PR Checklist" or "Coding Guidelines" in the Guidelines folder.
   - If found, read them using \`docs(method: "get", projectId: "{project_id}", documentId: "<doc-id>")\`.
   - You MUST enforce these team conventions rigorously during your review.

## Phase 4 — Review Checklist

7. Evaluate the implementation against the team guidelines AND this standard checklist:

   **Requirements Coverage:**
   - [ ] All acceptance criteria from the Story description are addressed
   - [ ] Edge cases mentioned in technical notes are handled
   - [ ] No acceptance criteria was skipped without explanation

   **Team Guidelines Compliance:**
   - [ ] Adheres to all rules in the "Coding Guidelines" document (if found)
   - [ ] Passes all items in the team "PR Checklist" (if found)

   **Implementation Quality:**
   - [ ] Implementation comment exists with clear summary
   - [ ] Branch name follows convention: \`feat/{issue-code}-*\`
   - [ ] Tests were mentioned and pass

   **Documentation:**
   - [ ] If code changes affect docs, were docs updated?
   - [ ] If new API endpoints were added, is there documentation?

   **Completeness:**
   - [ ] No TODO comments left for critical functionality
   - [ ] Dependencies on other Stories are noted if incomplete

## Phase 5 — Decision

8. Based on the checklist, make one of three decisions:

   **APPROVED ✅** — All acceptance criteria met, tests pass, documentation current.
   - Post approval comment
   - Tell the human: "This issue is ready to be moved to Done."
   - Do NOT move the status yourself — only humans can promote to Done.

   **CHANGES REQUESTED 🔄** — Some criteria not met, but fixable.
   - Post detailed feedback comment
   - Move back to In Progress.

   **BLOCKED ⛔** — Cannot review (missing context, dependency not met).
   - Post blocked comment explaining why
   - Do NOT change the status

## Phase 6 — Post Review Comment

9. Call \`issues(method: "comment", ...)\` with structured feedback using the appropriate template (Approved or Changes Requested).

## Phase 7 — Handle Follow-Up Comments

If the human leaves **additional comments after your review**, re-enter this workflow:

10. Re-read comments via \`issues(method: "get", ...)\`.

11. Identify new comments posted after your last review comment.

12. For each new comment, follow the same classification from Phase 2:
    - **Question** → Answer it
    - **Minor Fix** → Fix, test, post update comment
    - **Discussion** → Respond and wait
    - **Disagreement with review** → Re-evaluate your finding, post revised assessment

13. If the human's comments change your decision:
    - Update your review comment with a revised decision
    - If previously rejected but now satisfied → move back to In Review (if in In Progress)
    - If previously approved but new concerns raised → post a revised review

## Constraints
- 🚫 NEVER promote an issue to "Done" — only humans can do this
- 🚫 NEVER approve without checking acceptance criteria from the parent Story
- 🚫 NEVER ignore human comments — every comment must be acknowledged
- 🚫 NEVER treat a question as a rejection — respond first, then reassess
- ✅ Always provide specific, actionable feedback
- ✅ Always respond to every human comment before making a decision
- ✅ Always move rejected issues back to In Progress with clear next steps
---
_Integrity: 138 lines · workflow:review-task · DO NOT MODIFY_