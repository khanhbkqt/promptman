---
description: Detect project state (new/early/stable), verify document structure and guidelines, suggest improvements.
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
# Workflow: Project Health Check

Analyze the current project state — issues, documents, guidelines — and suggest
improvements. Run after `setup-rules` on first connection, or periodically to audit project health.

## Inputs
- **project_id** — The VibePM project ID or code

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- \`<toolname>(method: "help")\` — list all available commands
- \`<toolname>(method: "help", command: "<name>")\` — show parameters for a specific command

This works for all tools: \`issues\`, \`docs\`, \`projects\`, \`search\`, \`reports\`, \`notification\`.

## Phase 1 — Gather Project Info

1. Call \`projects(method: "get", projectId: "{project_id}")\` to retrieve:
   - Project details and statuses
   - Project code for display

2. Call \`reports(method: "project_health", projectId: "{project_id}")\` to check:
   - Total issue count and types present
   - Project health metrics and completion
   - Call \`reports(method: "overdue_items", projectId: "{project_id}")\` for overdue items.

3. Call \`docs(method: "list", projectId: "{project_id}")\` to check:
   - Total document count
   - Whether documents have parent-child hierarchy
   - Whether a "Guidelines" or "Coding Guidelines" document exists

## Phase 2 — Classify Project State

Use these signals to determine the project state:

| Signal | 🆕 New (empty) | 🌱 Early-stage | 🏗️ Stable |
|--------|----------------|----------------|-----------|
| Issues | 0 | Few, limited types | Many, diverse types |
| Docs | 0 | Some, flat structure | Hierarchical with folders |
| Guidelines doc | None | None | Present |

## Phase 3 — Report & Suggest

Present your findings to the user and suggest actions based on the state.

### 🆕 New Project (0 issues, 0 docs)

Report:
\`\`\`
📊 Project Health: {projectCode}

State: 🆕 New (empty project)
Issues: 0
Documents: 0

Recommendations:
1. Create document structure (Specs/, Design/, Architecture/, Guidelines/, Meeting Notes/)
2. Create a Coding Guidelines template in Guidelines/
3. Start planning — use the brainstorm or plan-feature workflow

Shall I create the document structure now?
\`\`\`

If user confirms, create the folder documents:
- \`docs(method: "create", title: "Specs", brief: "Feature specifications and requirements")\`
- \`docs(method: "create", title: "Design", brief: "Architecture decisions, API design, database schemas")\`
- \`docs(method: "create", title: "Architecture", brief: "System-level documentation, tech stack, deployment")\`
- \`docs(method: "create", title: "Guidelines", brief: "Team conventions, checklists, and coding standards")\`
- \`docs(method: "create", title: "Meeting Notes", brief: "Brainstorm results, sprint planning notes")\`

### 🌱 Early-stage Project (some issues/docs, no guidelines)

Report:
\`\`\`
📊 Project Health: {projectCode}

State: 🌱 Early-stage
Issues: {count} ({types list})
Documents: {count} (hierarchy: {yes/no})

Missing:
- ❌ No Coding Guidelines document found
- ❌ Documents not organized in folders (optional)

Recommendations:
1. Create a Coding Guidelines doc so agents can self-review during Quality Gate
2. Consider organizing docs under folder parents (Specs/, Design/, etc.)

Shall I create a Guidelines folder and template?
\`\`\`

### 🏗️ Stable Project (many issues/docs, guidelines exist)

Report and automatically read the guidelines:
\`\`\`
📊 Project Health: {projectCode}

State: 🏗️ Stable
Issues: {count} ({types list})
Documents: {count} (hierarchical: ✅)
Guidelines: ✅ Found "{guidelines doc title}"

✅ Read project guidelines and conventions. Ready to work.
\`\`\`

Then call \`docs(method: "get", ...)\` on the Guidelines document and append key rules
to your local AGENTS.md under a \`## Project Conventions\` section.

## Phase 4 — Record Conventions Locally

If a Guidelines document was found:
1. Read it via \`docs(method: "get", projectId: "{project_id}", documentId: "<guidelines-id>")\`
2. Append the key rules to your local \`AGENTS.md\` under \`## Project Conventions\`
3. This ensures you follow team conventions in all subsequent work

## Constraints
- 🚫 NEVER create documents without user confirmation (for 🆕 and 🌱 states)
- 🚫 NEVER skip reading the Guidelines doc if one exists (for 🏗️ state)
- ✅ ALWAYS present findings before taking action
- ✅ ALWAYS suggest the complete-task workflow uses Guidelines for quality gate

## Summary & Next Steps

When this workflow completes, present based on project state:

**🆕 New Project:**
\\\`\\\`\\\`
📊 Project health check complete: {projectCode} — 🆕 New project.

🔜 Suggested next steps:
   → Create document structure (if approved)
   → Use the **brainstorm** workflow to start planning features
\\\`\\\`\\\`

**🌱 Early-stage or 🏗️ Stable:**
\\\`\\\`\\\`
📊 Project health check complete: {projectCode} — {state}.
📋 Issues: {count}, Documents: {count}, Guidelines: {yes/no}

🔜 Suggested next steps:
   → Use the **check-notifications** workflow to see pending work
   → Use the **start-task** workflow if ready to begin implementation
   → Use the **plan-scope** workflow if milestones need planning
\\\`\\\`\\\`
---
_Integrity: 146 lines · workflow:project-health · DO NOT MODIFY_
