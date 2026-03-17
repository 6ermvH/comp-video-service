# Video Comparison Service

Platform for controlled pairwise video comparison (baseline vs candidate) for flooding/explosion VFX research. Participants evaluate video pairs; administrators manage studies and analyze results.

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25 + Gin |
| Frontend | React 19 + Vite |
| Database | PostgreSQL 16 |
| Video storage | MinIO (S3-compatible) |
| Containers | Docker Compose |
| CI | GitHub Actions |

---

## Quick Start

### 1. Clone and configure

```bash
git clone <repo-url>
cd comp-video-service
cp .env.example .env
```

Edit `.env` — at minimum set a strong JWT secret:
```
JWT_SECRET=your_random_secret_at_least_32_chars
```

### 2. Start all services

```bash
docker compose up -d --build
```

Services start in order: PostgreSQL → MinIO → Backend → Frontend.
Migrations are applied automatically on backend startup.

### 3. Create the first admin user

```bash
docker compose run --rm \
  -e SEED_USERNAME=admin \
  -e SEED_PASSWORD=yourpassword \
  seed
```

Or uncomment the `seed` service in `docker-compose.yml` and set `SEED_USERNAME` / `SEED_PASSWORD` in `.env`.

### 4. Open in browser

| URL | Purpose |
|---|---|
| http://localhost:5173 | Participant interface |
| http://localhost:5173/admin/login | Admin panel |
| http://localhost:9001 | MinIO Console |

---

## Architecture

```
browser
  ├── :5173  →  Vite dev server (frontend SPA)
  │               └── /api/* proxy → backend:8080
  └── :9000  →  MinIO (direct video GET, public bucket)

backend:8080
  ├── /api/session/*   — participant (no auth)
  ├── /api/task/*      — participant (no auth)
  └── /api/admin/*     — admin (JWT + CSRF)

postgres:5432  ←  backend (internal Docker network)
minio:9000     ←  backend (upload), browser (download)
```

---

## Admin Workflow

### 1. Create a study
`/admin/studies` → **+ Create**

Fields: name, effect type (`flooding` / `explosion` / `mixed`), tasks per participant, instructions text, options (tie allowed, reasons, confidence).

### 2. Create groups
`/admin/pairs` → select study → **Groups** section → **+ Group**

A group is a category of pairs (e.g. "scene 1", "urban location"). Copy the group UUID — you'll need it for CSV import.

### 3. Import pairs via CSV

File format:
```csv
group_id,source_image_id,pair_code,difficulty,is_attention_check,notes,baseline_s3_key,candidate_s3_key
550e8400-...,img_001,flood_001,easy,false,description,,
550e8400-...,img_002,flood_002,medium,true,attention check,,
```

| Column | Required | Description |
|---|---|---|
| `group_id` | **yes** | Group UUID |
| `source_image_id` | no | Your internal image ID |
| `pair_code` | no | Human-readable code |
| `difficulty` | no | `easy` / `medium` / `hard` |
| `is_attention_check` | no | `true` / `false` |
| `notes` | no | Comment |
| `baseline_s3_key` | no | S3 key of an already-uploaded video |
| `candidate_s3_key` | no | S3 key of an already-uploaded video |

### 4. Upload video assets
`/admin/pairs` → **Upload video asset**

Each pair needs two videos: `baseline` and `candidate`. A pair is only assigned to participants once both assets are present.

### 5. Activate the study
`/admin/studies` → **→ Active**

Only active studies are accessible to participants.

### 6. Share the link
`/admin/studies` → **🔗 Link** — copies a URL like:
```
http://localhost:5173/?study_id=<uuid>
```

---

## Participant Workflow

```
Welcome → Instructions → Practice → Tasks → Break → Completion
```

1. Open the study link
2. Fill in role and experience
3. Read instructions
4. Complete practice pairs
5. Rate video pairs: choose **A** / **B** / **Tie**, select reason and confidence
6. Receive completion code

### Keyboard shortcuts on the task page

| Key | Action |
|---|---|
| `1` | Select left video (A) |
| `2` | Select right video (B) |
| `0` | Tie |
| `R` | Replay both videos |
| `N` | Next task (after selection) |

---

## REST API

OpenAPI spec: `backend/docs/swagger.yaml`

### Public endpoints (participant)

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/session/start` | Start session → `session_token` + first task |
| `GET` | `/api/session/:token/next-task` | Next task (204 if no tasks left) |
| `POST` | `/api/session/:token/complete` | Complete → `completion_code` |
| `POST` | `/api/task/:id/response` | Submit response |
| `POST` | `/api/task/:id/event` | Log event (replay, etc.) |

### Admin endpoints (JWT + CSRF)

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/admin/login` | Login → JWT + CSRF token |
| `GET` | `/api/admin/studies` | List studies |
| `POST` | `/api/admin/studies` | Create study |
| `PATCH` | `/api/admin/studies/:id` | Update status |
| `GET` | `/api/admin/studies/:id/groups` | List groups |
| `POST` | `/api/admin/studies/:id/groups` | Create group |
| `POST` | `/api/admin/studies/:id/import` | CSV pair import |
| `POST` | `/api/admin/assets/upload` | Upload video asset (MP4) |
| `GET` | `/api/admin/source-items` | List pairs (filters: study_id, group_id) |
| `GET` | `/api/admin/analytics/overview` | Summary analytics |
| `GET` | `/api/admin/analytics/study/:id` | Per-study details |
| `GET` | `/api/admin/analytics/qc` | Participant QC report |
| `GET` | `/api/admin/export/csv` | Export responses as CSV |
| `GET` | `/api/admin/export/json` | Export responses as JSON |

---

## Project Structure

```
comp-video-service/
├── backend/
│   ├── cmd/
│   │   ├── server/main.go        # Entrypoint, router, DI
│   │   └── seed/main.go          # Admin user seeder CLI
│   ├── internal/
│   │   ├── config/               # Env-based config
│   │   ├── handler/              # Gin HTTP handlers
│   │   ├── middleware/           # JWT, CSRF, rate limiting
│   │   ├── model/                # Domain structs
│   │   ├── repository/           # PostgreSQL queries (pgx/v5)
│   │   ├── service/              # Business logic
│   │   └── storage/              # S3/MinIO client
│   ├── migrations/               # golang-migrate SQL files
│   ├── tests/                    # Integration tests (build tag: integration)
│   └── docs/                     # OpenAPI spec (swagger.yaml, swagger.json)
├── frontend/
│   └── src/
│       ├── api/client.js         # All HTTP calls — single source of truth
│       ├── context/              # SessionContext, ToastContext
│       ├── components/           # SyncVideoPlayer, ChoicePanel, etc.
│       └── pages/                # Participant and admin pages
├── .github/workflows/
│   ├── backend.yml               # Lint, unit tests, coverage ≥70%, build
│   ├── frontend.yml              # Lint, build
│   └── api-contract.yml         # OpenAPI spec drift + frontend contract check
├── docs/
│   └── to_contributor.md         # Development workflow with LLM agents
├── AGENTS.md                     # Project context for all agents
├── INFRA_AGENT.md                # Instructions for Agent 1 (infra)
├── BACKEND_AGENT.md              # Instructions for Agent 3 (backend)
├── FRONTEND_AGENT.md             # Instructions for Agent 2 (frontend)
├── docker-compose.yml
├── .env.example
└── sample_pairs.csv              # Example CSV for pair import
```

---

## Environment Variables

| Variable | Example | Description |
|---|---|---|
| `POSTGRES_DB` | `comp_video` | Database name |
| `POSTGRES_USER` | `postgres` | Database user |
| `POSTGRES_PASSWORD` | `postgres` | Database password |
| `MINIO_ROOT_USER` | `minioadmin` | MinIO login |
| `MINIO_ROOT_PASSWORD` | `minioadmin` | MinIO password |
| `S3_BUCKET` | `videos` | Bucket name |
| `S3_PUBLIC_URL` | `http://localhost:9000` | Public MinIO URL (seen by browser) |
| `BACKEND_PORT` | `8080` | Backend port |
| `JWT_SECRET` | — | JWT secret (**must be changed**) |
| `CORS_ORIGINS` | `http://localhost:5173` | Allowed origins |
| `SEED_USERNAME` | `admin` | Admin username for seeder |
| `SEED_PASSWORD` | — | Admin password for seeder (**required**) |

---

## Development

### Backend (local, no Docker)

```bash
cd backend
go mod download
go run ./cmd/server
```

### Frontend (local, no Docker)

```bash
cd frontend
npm install
npm run dev   # http://localhost:5173
```

### Tests

```bash
# Backend unit tests + coverage
cd backend && go test ./... -race -count=1

# Backend integration tests (requires Docker)
cd backend && go test -tags integration ./tests/... -v

# Backend linter
cd backend && golangci-lint run ./...

# Frontend linter
cd frontend && npm run lint

# Frontend build
cd frontend && npm run build

# API contract validation
cd frontend && npm run validate:api
```

### Regenerate OpenAPI spec

```bash
cd backend
swag init -g cmd/server/main.go -o docs --outputTypes yaml,json --quiet
```

Commit the updated `docs/swagger.yaml` and `docs/swagger.json`.

---

## CI

| Workflow | Triggers | What it checks |
|---|---|---|
| `backend.yml` | `backend/**` | lint, unit tests, coverage ≥70%, binary builds, integration tests |
| `frontend.yml` | `frontend/**` | lint, production build |
| `api-contract.yml` | `backend/**`, `frontend/src/api/**` | spec not drifted, all frontend API calls exist in spec |

---

## Working with LLM Agents

This project is developed using LLM coding agents. See [docs/to_contributor.md](docs/to_contributor.md) for how to initialize and use them.
