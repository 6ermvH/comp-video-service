# Working with LLM Agents

This project is actively developed using LLM coding agents. Each agent has a defined scope, a dedicated instruction file, and a task handoff protocol.

---

## Agent Roles

| Agent | Scope | Instruction file |
|-------|-------|-----------------|
| **Agent 1** — Infra | CI/CD, Docker, docker-compose, task authoring, GitHub Issues | `INFRA_AGENT.md` |
| **Agent 2** — Frontend | `frontend/src/`, `frontend/scripts/`, `frontend/package.json` | `FRONTEND_AGENT.md` |
| **Agent 3** — Backend | `backend/internal/`, `backend/cmd/`, `backend/migrations/` | `BACKEND_AGENT.md` |

Agents do not cross their scope boundaries. A backend bug fix goes to Agent 3 only; a UI change goes to Agent 2 only. Agent 1 orchestrates and formulates tasks.

---

## How to Initialize an Agent

Each agent reads its instruction file at the start of a session. The instruction file contains:
- the agent's role and areas of responsibility
- code conventions and patterns used in this project
- how to add new code and how to refactor safely
- quality standards and definition of done

### Claude Code (Agent 1 — Infra)

Open Claude Code in the repo root. It automatically loads `AGENTS.md` as project context.
For Agent 1-specific rules, paste the contents of `INFRA_AGENT.md` into the first message or reference it:

```
Read INFRA_AGENT.md and follow its rules for this session.
```

### Claude Code or Codex (Agent 2 — Frontend)

Start a new session and paste the contents of `FRONTEND_AGENT.md` as the system prompt or first message:

```
Read FRONTEND_AGENT.md and follow its rules for this session.
Then read the task file I will give you.
```

Then paste or attach the relevant `tasks_agent2_<topic>.md` file.

### Claude Code or Codex (Agent 3 — Backend)

Same pattern with `BACKEND_AGENT.md`:

```
Read BACKEND_AGENT.md and follow its rules for this session.
Then read the task file I will give you.
```

Then paste or attach the relevant `tasks_agent3_<topic>.md` file.

---

## Task File Protocol

Agent 1 writes task files for Agents 2 and 3. Task files live in the repo root:

```
tasks_agent2_<topic>.md   — for Agent 2 (frontend)
tasks_agent3_<topic>.md   — for Agent 3 (backend)
```

### Task file structure

```markdown
# Agent N Tasks — <Topic>

## Context
Why this is needed. What problem is being solved. What to watch out for.

---

## Task 1 — <Title>

**File:** `path/to/file`

What to add / change / remove. Include ready code when helpful.

---

## Task 2 — ...

## Verification

Commands to confirm the work is correct.
```

### Rules

- **Context is required.** Agents have no memory across sessions.
- **Exact file paths.** Not "somewhere in handlers" — `backend/internal/handler/admin.go`.
- **Ready code when non-obvious.** Not "add a field", show the struct.
- **One topic per file.** Don't mix unrelated changes.
- **State what not to touch.** Especially important for refactoring tasks.

---

## General Workflow

```
1. Agent 1 diagnoses the problem or designs a feature
2. Agent 1 writes tasks_agent{2|3}_<topic>.md
3. Agent 2 or Agent 3 reads the task file and implements
4. Agent runs verification commands (lint, test, build)
5. Agent 1 reviews output, creates follow-up tasks if needed
```

---

## Project Context File

`AGENTS.md` in the repo root is a shared context file read by all agents. It contains:
- project overview and stack
- directory structure with key files annotated
- backend and frontend conventions
- infrastructure details (Docker ports, env vars, seed)
- CI pipeline summary
- common pitfalls

Keep `AGENTS.md` up to date when making architectural changes.

---

## Tips

- Always read the instruction file (`*_AGENT.md`) **before** the task file — it establishes conventions the task assumes.
- If a task references a file the agent hasn't read yet, instruct it to read the file first before making changes.
- For large refactors, split into multiple task files — one structural change, then one behavioral change.
- After an agent session, verify with the checklist in the relevant `*_AGENT.md` (Definition of Done or Pre-completion Checklist).
