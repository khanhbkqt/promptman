---
description: Install all VibePM skills, workflows, and IDE rules as **local files** in the project. After this, you work from local files — no need to read MCP resources during normal work. Re-run this workflow to pull updates.
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
# Workflow: Setup Rules

Install all VibePM skills, workflows, and IDE rules as **local files** in the project.
After this, you work from local files — no need to read MCP resources during normal work.
Re-run this workflow to pull updates.

## Inputs
- None required. You are a unified VibePM Agent.

## Tool Discovery

If unsure about a tool's available commands or parameters, use the built-in help:

- `<toolname>(method: "help")` — list all available commands
- `<toolname>(method: "help", command: "<name>")` — show parameters for a specific command

This works for all tools: `issues`, `docs`, `projects`, `search`, `reports`, `notification`.

## Phase 1 — Fetch Everything from MCP

Download all resources you need in one batch:

1. **Agent skill** — `read_resource("vibepm://agent-skills/vibepm-agent")`
2. **Topic skills** — Fetch ALL of these:
   - `read_resource("vibepm://agent-skills/vibepm-data-model")`
   - `read_resource("vibepm://agent-skills/vibepm-red-lines")`
   - `read_resource("vibepm://agent-skills/vibepm-documents")`
   - `read_resource("vibepm://agent-skills/vibepm-planning")`
   - `read_resource("vibepm://agent-skills/vibepm-task-execution")`
   - `read_resource("vibepm://agent-skills/vibepm-review")`
3. **Workflows** — Use `list_resources` to discover all `vibepm://workflows/*` resources, then `read_resource` each one.

## Phase 2 — Install Locally

**CRITICAL — ANTI-COMPRESSION RULES:**
- You MUST write every resource **exactly as received** — character for character.
- Do NOT summarize, compress, rephrase, shorten, or "improve" any content.
- Do NOT rewrite steps in your own words or merge steps together.
- Do NOT drop code blocks, tables, examples, or "obvious" lines.
- Each workflow includes an **integrity footer** with a line count. After writing,
  verify the output file line count matches. If it doesn't, you compressed the content.
- **If you find yourself thinking "this is too verbose, I'll simplify" — STOP. Write it verbatim.**

### Step 1: Write AGENTS.md (Universal — Always Do This)

Write the **agent skill** content to `AGENTS.md` in the project root.
This is the universal standard that any AI IDE will auto-load.

### Step 2: Write All Skills to .agents/skills/

Create a skill directory for EACH skill and write it:

```
.agents/skills/vibepm-agent/SKILL.md            ← unified agent skill (same as AGENTS.md)
.agents/skills/vibepm-data-model/SKILL.md       ← data model guide
.agents/skills/vibepm-red-lines/SKILL.md        ← backend rules
.agents/skills/vibepm-documents/SKILL.md        ← document management
.agents/skills/vibepm-planning/SKILL.md         ← feature planning
.agents/skills/vibepm-task-execution/SKILL.md   ← task lifecycle
.agents/skills/vibepm-review/SKILL.md           ← review workflow
```

Create directories if they don't exist. Each SKILL.md must include the
YAML frontmatter (`---\nname: ...\ndescription: ...\n---`) from the source.

### Step 3: Write All Workflows to .agents/workflows/

For EVERY workflow fetched in Phase 1, write it to:
```
.agents/workflows/{workflow-name}.md
```

Ensure each file includes YAML frontmatter:
```
---
description: <workflow description>
---
<workflow content>
```

### Step 4: IDE-Specific Rules (Optional, If Detected)

If you detect a specific IDE, also write the agent skill to the IDE-specific location:

| Environment | Directory Path | Filename |
|---|---|---|
| **Cursor** | `.cursor/rules/` | `vibepm-agent.mdc` |
| **Windsurf** | `.windsurf/rules/` | `vibepm-agent.md` |
| **Roo Code / Cline** | `.roo/rules/` or `.clinerules/` | `vibepm-agent.md` |
| **Trae** | `.trae/rules/` | `vibepm-agent.md` |
| **GitHub Copilot** | `.github/instructions/` | `vibepm-agent.md` |
| **Claude Code** | `CLAUDE.md` (project root) | — |

## Phase 3 — Verify & Confirm

Read back `AGENTS.md` and one skill file to confirm writes succeeded.

Report to the user:
```
✅ VibePM agent installed
📄 AGENTS.md — agent entrypoint
📚 Installed {N} skills to .agents/skills/
⚙️ Installed {M} workflows to .agents/workflows/
🎯 IDE-specific: {path} (if applicable)
🔄 Re-run this workflow anytime to pull updates from VibePM.
```

## Error Recovery
- If `read_resource` fails: check your MCP connection.
- If file write fails: ensure directories exist before writing, check filesystem permissions.
---
_Integrity: 110 lines · workflow:setup-rules · DO NOT MODIFY_
