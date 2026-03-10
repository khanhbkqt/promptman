---
description: Define the scope of a milestone or phase: review brainstorm outputs, group features, set boundaries, and produce a scoping document with human approval.
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
# Workflow: Plan Scope

Define the scope of a milestone or phase: review brainstorm outputs, group features,
set boundaries, and produce a scoping document with human approval.

## Inputs
- **project_id** — The VibePM project ID or code
- **scope_description** — What this milestone aims to deliver (e.g. "Reader MVP", "Payment integration phase")
- **milestone_name** — (optional) Name for the milestone (e.g. "Phase 1a: Reader MVP")

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Gather Inputs

1. Call `docs(method: "list", projectId: "{project_id}")` and look for:
   - Brainstorm documents (titles starting with "Brainstorm:")
   - Product vision / PRD documents
   - Previous milestone/scope documents
   Read relevant ones with `docs(method: "get", ...)`.

2. Call `issues(method: "list", projectId: "{project_id}", type: "EPIC")` to see existing Epics.
   - Which are already planned but not started?
   - Which are in progress?

3. Call `issues(method: "list", projectId: "{project_id}", type: "STORY")` to see existing Stories.
   - Any orphan Stories (no Epic) that relate to this scope?

## Phase 2 — Define Scope

4. Based on the inputs, define the milestone scope:

   **Inclusion criteria — features belong in this milestone if they:**
   - Deliver user-visible value toward the stated goal
   - Have clear enough requirements to plan
   - Don't have unresolvable external dependencies

   **Exclusion criteria — features do NOT belong if they:**
   - Are nice-to-have but not essential for the stated goal
   - Require major research or unknowns
   - Depend on other milestones completing first

5. Organize included features into **phases** (if the milestone is large):
   - **Phase A:** Foundation / must-have features
   - **Phase B:** Enhancement features (build on Phase A)
   - **Phase C:** Polish / optimization

6. For each feature, note:
   - Whether an Epic already exists or needs to be created
   - Rough size: S / M / L / XL
   - Dependencies on other features within the milestone

## Phase 3 — Risk Assessment

7. Identify risks:
   - **Technical risks** — unknown tech, complex integrations
   - **Scope risks** — features that might expand beyond estimates
   - **Dependency risks** — external services, other teams, or milestones
   - **Timeline risks** — features on the critical path

## Phase 4 — Create Scoping Document (CHECKPOINT)

8. Create a VibePM document:

   ```
   docs(method: "create",
     projectId: "{project_id}",
     title: "{milestone_name or 'Scope: ' + scope_description}",
     content: "<structured content>"
   )
   ```

   **Document Template:**
   ```markdown
   # {milestone_name}

   **Status:** Draft — awaiting approval
   **Goal:** <1-2 sentence summary of what "done" looks like>

   ## In Scope

   ### Phase A: <Foundation>
   | # | Feature | Size | Epic Exists? | Depends On |
   |---|---------|------|-------------|------------|
   | 1 | <feature> | M | ✅ PRJ-10 | None |
   | 2 | <feature> | L | ❌ Needs creation | #1 |

   ## Out of Scope
   - <Feature X> — Reason: <why excluded>

   ## Risks
   - ⚠️ <risk description> — Mitigation: <how to handle>

   ## Success Criteria
   - [ ] <measurable outcome 1>
   - [ ] <measurable outcome 2>

   ## Next Steps
   - [ ] Human approves scope
   - [ ] Link this document to Epics using \`issues(method: "update", ...)\` (\`linkDocumentIds\`)
   - [ ] Use the **plan-feature** workflow for each feature needing Epics
   - [ ] Use the **sprint-planning** workflow to populate the first board
   ```

9. Present the scope to the human and **STOP.** Wait for human approval.

## Constraints
- 🚫 Do NOT create Epics or Stories during scoping — this is planning, not execution
- 🚫 Do NOT skip the risk assessment
- ✅ Always separate features into phases with clear dependencies
- ✅ Always define explicit "Out of Scope" to prevent scope creep
---
_Integrity: 117 lines · workflow:plan-scope · DO NOT MODIFY_
