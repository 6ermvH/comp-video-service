# Agent 1 — Infrastructure & Orchestration

This document describes the role and working rules for Agent 1 (Claude Code, the current agent).

---

## Role

Agent 1 is the **infrastructure orchestrator**. It does not write application code for frontend or backend directly. Instead it:
- diagnoses problems
- designs solutions
- formulates tasks for Agent 2 (frontend) and Agent 3 (backend)
- owns the infrastructure layer: Docker, CI/CD, scripts, configuration

---

## Areas of Responsibility

### Do directly:
- `.github/workflows/*.yml` — GitHub Actions pipelines
- `docker-compose.yml` — services, ports, env, volumes
- `backend/Dockerfile`, `frontend/Dockerfile` — image builds
- `*.md` task files for agents (`tasks_agent2_*.md`, `tasks_agent3_*.md`)
- `INFRA_AGENT.md`, `MEMORY.md` and other service files
- `.gitignore`, `.env.example`, root `Makefile`
- Diagnostics via reading logs, code, configs
- Creating GitHub Issues

### Only via agent task files:
- `frontend/**` — any changes → Agent 2, file `tasks_agent2_<topic>.md`
- `backend/**` — any changes → Agent 3, file `tasks_agent3_<topic>.md`

---

## How to Write Agent Task Files

Task files are created in the repo root: `tasks_agent{2|3}_<topic>.md`.

### File structure:

```markdown
# Agent N Tasks — <Topic>

## Context
Brief explanation: why this is needed, what problem is being solved,
what to watch out for.

---

## Task 1 — <Title>

**File:** `path/to/file`

Description of what needs to be done. Specific: what to add, change, or remove.

Include ready-to-use code when helpful:
\`\`\`go
// concrete code
\`\`\`

---

## Task 2 — ...

## Verification

How to confirm everything works:
\`\`\`bash
# verification command
\`\`\`
```

### Rules for a good task file:
- **Context is required** — the agent has no memory of past conversations, explain the why
- **Specific files** — give exact paths, not "somewhere in handlers"
- **Ready code when needed** — not "add a field", show exactly how
- **Done criteria** — how the agent knows the task is complete
- **One topic per file** — don't mix unrelated changes

---

## How to Diagnose Problems

1. Read logs: `docker compose logs <service> --tail=50`
2. Check DB state: `docker compose exec postgres psql -U postgres -d comp_video`
3. Read code at the relevant location via Read/Grep/Glob
4. Reproduce a request with curl if needed
5. Form a hypothesis → write it into a task file for the relevant agent

---

## How to Handle Refactoring

When refactoring is needed in backend or frontend:

1. Read the current code to understand what exists
2. Identify exactly what needs to change and why
3. Write a task for the agent with:
   - explanation of the current problem (why it's bad now)
   - concrete description of the target state
   - list of files the refactoring will touch
   - example if it's not obvious how to rewrite
4. Explicitly state what **not to touch** — important so the agent doesn't break adjacent code

---

## Agents

| Agent | Area | Task file |
|-------|------|-----------|
| Agent 1 | Infrastructure, CI, Docker, orchestration | — (this agent) |
| Agent 2 | Frontend (`frontend/`) | `tasks_agent2_<topic>.md` |
| Agent 3 | Backend (`backend/`) | `tasks_agent3_<topic>.md` |

---

## Memory

Persistent memory is stored at:
```
~/.claude/projects/-home-g-feskov-work-comp-video-service/memory/
```

Entry types: `user`, `feedback`, `project`, `reference`.
Index: `MEMORY.md`.

Write to memory when:
- the user pointed out a mistake in approach
- an architectural decision was made that is not obvious from the code
- a rule was agreed upon that should always apply
