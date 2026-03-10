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
# Workflow: Plan Feature

Decompose a feature into a well - structured Epic → Stories hierarchy.
Each Story is a shippable deliverable with comprehensive task breakdown instructions for Member agents.

## Inputs
   - ** project_id ** — The VibePM project ID or code
      - ** feature_description ** — Description of the feature to plan

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Project Context

1. Call `projects(method: "get", projectId: "{project_id}")` to retrieve:
   - Available statuses → find the "Backlog" status ID
   - Project code → needed for issue code references

2. Call `issues(method: "list", projectId: "{project_id}", type: "EPIC")` to check for existing related Epics.
   - If a closely related Epic already exists, consider adding Stories to it instead.
   - **CHECKPOINT**: If you find a related Epic, ask the human: "Should I add Stories to it, or create a new Epic?"

3. Use document tools to retrieve the project's documentation.
   - **CHECKPOINT**: List all related documents and ask the human which to reference.

## Phase 2 — Design the Hierarchy

4. Before creating anything, design the full hierarchy mentally:

   **Decomposition principles:**
   - Each Story should be **independently shippable** — delivers user-visible value on its own
   - Stories should be **roughly equal in size**
   - Order Stories by **dependency** — dependent Stories come first
   - Aim for **2-4 Stories per Epic**

   **Anti-patterns to avoid:**
   - ❌ Stories that are just "Backend" and "Frontend" splits
   - ❌ Stories that can't be demoed to a stakeholder
   - ❌ Stories with circular dependencies
   - ❌ Vague Stories like "Set up infrastructure"

## Phase 3 — Create the Epic

5. Call `issues(method: "create", ...)` to create the Epic:
   ```
   issues(method: "create",
     projectId: "{project_id}", type: "EPIC",
     title: "<Clear, action-oriented title>",
     statusId: "<backlog-status-id>",
     description: "<2-3 sentence summary of the feature, its business value, and success criteria>"
   )
   ```
   Save the returned Epic ID.

   - **IMPORTANT**: Link relevant documents using `issues(method: "update", ...)` with `linkDocumentIds`.

## Phase 4 — Create Stories with Task Breakdown

6. For each Story, call `issues(method: "create", ...)` with a **comprehensive description** using this template:

   ```markdown
   ## Overview
   <1-2 sentences: what this Story delivers and why it matters>

   ## Acceptance Criteria
   - [ ] <Specific, testable criterion>
   - [ ] <Include edge cases>
   - [ ] <Performance requirements if applicable>

   ## Suggested Task Breakdown
   1. **<Task title>** — <1 sentence: what this task implements>
      - Files likely affected: <list key files/modules>
      - Estimated complexity: Small / Medium / Large
   2. **<Task title>** — <1 sentence>
      - Files likely affected: <list>
      - Estimated complexity: Small / Medium / Large

   ## Technical Notes
   - <Architecture decisions or patterns to follow>
   - <Existing code to reference or extend>

   ## API Contract (if applicable)
   - Endpoint: \`METHOD /api/path\`
   - Request: \`{ field: type }\`
   - Response: \`{ field: type }\`

   ## Dependencies
   - <Other Story codes that must complete first, or "None">

   ## Out of Scope
   - <What this Story explicitly does NOT cover>
   ```

7. **Quality check** each Story description before creating:
   - Does every acceptance criterion have a clear pass/fail test?
   - Does the task breakdown cover all acceptance criteria?
   - Are technical notes specific enough for a Member to start without asking questions?

## Phase 5 — Summary

8. Present the complete hierarchy to the human.

## Constraints
- 🚫 Do NOT create Tasks — Member agents own task breakdown
- 🚫 Do NOT create Stories from vague input — ask for clarification
- ✅ All issues MUST be created in Backlog status
- ✅ Every Story MUST have all template sections filled out
---
_Integrity: 113 lines · workflow:plan-feature · DO NOT MODIFY_