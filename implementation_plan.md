# Video Comparison Service — Revised Implementation Plan (v2)

Платформа контролируемого парного сравнения видео (baseline vs candidate) для эффектов flooding/explosion.

> [!IMPORTANT]
> **Полностью переписывать проект НЕ нужно.** Инфраструктура (Docker, MinIO, PG, Go skeleton, React scaffold) сохраняется. Нужно расширить модель данных, добавить новые сущности, переделать flow респондента и расширить аналитику.

---

## Что сохраняется ✅

| Компонент | Файлы |
|---|---|
| Docker Compose (PG + MinIO + backend + frontend) | `docker-compose.yml` |
| Go project structure (cmd/server, internal/) | полностью |
| Config, DB connection, S3 client | `config/`, `storage/s3.go`, `repository/db.go` |
| JWT auth middleware + admin auth | `middleware/auth.go`, `handler/auth.go` |
| Admin model + repo | `model/admin.go`, `repository/admin.go` |
| Seed script | `cmd/seed/main.go` |
| React scaffold (Vite, routing, CSS tokens) | `frontend/` base |
| `.gitignore`, `.env.example`, Dockerfiles | root files |

## Что меняется ⚠️

| Компонент | Действие |
|---|---|
| DB schema (`001_init.up.sql`) | **Расширить** — добавить 6 новых таблиц |
| Models (Go structs) | **Расширить** — новые сущности, переделать Vote → Response |
| Repositories | **Расширить** — новые репо, переделать vote repo |
| Handlers (admin, comparison, vote) | **Значительная переработка** — новые endpoints, новая структура ответов |
| Services | **Значительная переработка** — randomization, session assignment, QC |
| Frontend pages | **~80% переписать** — 6 новых страниц, синхронизация видео, response panel |

---

## Обзор стека (без изменений)

| Слой | Технология |
|---|---|
| Backend | Go 1.24.4 + Gin |
| Frontend | React 19 (Vite) |
| Database | PostgreSQL 16 |
| Storage | S3 / MinIO |
| Container | Docker Compose |

---

## Новая схема БД

Существующие таблицы `admins` остаются без изменений.  
`videos` → переименована в `video_assets`, расширена.  
`comparisons` → переименована в `source_items` (пара видео от одного изображения).  
`votes` → заменена на `responses` с расширенной структурой.

```sql
-- ============================================================
-- 002_study_schema.up.sql
-- ============================================================

-- Исследование (flooding benchmark, explosion benchmark, etc.)
CREATE TABLE studies (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    VARCHAR(255) NOT NULL,
    effect_type             VARCHAR(20) NOT NULL,  -- flooding | explosion | mixed
    status                  VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft|active|paused|archived
    max_tasks_per_participant INTEGER NOT NULL DEFAULT 20,
    instructions_text       TEXT,
    tie_option_enabled      BOOLEAN NOT NULL DEFAULT true,
    reasons_enabled         BOOLEAN NOT NULL DEFAULT true,
    confidence_enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Группа внутри исследования (сцена / категория)
CREATE TABLE groups (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id            UUID NOT NULL REFERENCES studies(id),
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    priority            INTEGER NOT NULL DEFAULT 0,
    target_votes_per_pair INTEGER NOT NULL DEFAULT 10,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Исходный элемент (одно изображение → 2 видео)
CREATE TABLE source_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id        UUID NOT NULL REFERENCES studies(id),
    group_id        UUID NOT NULL REFERENCES groups(id),
    source_image_id VARCHAR(255),
    pair_code       VARCHAR(100),
    difficulty      VARCHAR(20),   -- easy | medium | hard
    is_attention_check BOOLEAN NOT NULL DEFAULT false,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Видео-ассет (baseline или candidate)
ALTER TABLE videos RENAME TO video_assets;
ALTER TABLE video_assets ADD COLUMN source_item_id UUID REFERENCES source_items(id);
ALTER TABLE video_assets ADD COLUMN method_type VARCHAR(20); -- baseline | candidate
ALTER TABLE video_assets ADD COLUMN width INTEGER;
ALTER TABLE video_assets ADD COLUMN height INTEGER;
ALTER TABLE video_assets ADD COLUMN fps REAL;
ALTER TABLE video_assets ADD COLUMN codec VARCHAR(50);
ALTER TABLE video_assets ADD COLUMN checksum VARCHAR(128);

-- Участник (один респондент)
CREATE TABLE participants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_token   VARCHAR(128) NOT NULL UNIQUE,
    study_id        UUID NOT NULL REFERENCES studies(id),
    device_type     VARCHAR(50),
    browser         VARCHAR(100),
    role            VARCHAR(50),        -- general_viewer | ml_practitioner | etc.
    experience      VARCHAR(50),        -- none | limited | moderate | strong
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    quality_flag    VARCHAR(20) DEFAULT 'ok'  -- ok | suspect | flagged
);

-- Представление пары конкретному участнику (с рандомизацией)
CREATE TABLE pair_presentations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id      UUID NOT NULL REFERENCES participants(id),
    source_item_id      UUID NOT NULL REFERENCES source_items(id),
    left_asset_id       UUID NOT NULL REFERENCES video_assets(id),
    right_asset_id      UUID NOT NULL REFERENCES video_assets(id),
    left_method_type    VARCHAR(20) NOT NULL,  -- baseline | candidate
    right_method_type   VARCHAR(20) NOT NULL,
    task_order          INTEGER NOT NULL,
    is_attention_check  BOOLEAN NOT NULL DEFAULT false,
    is_practice         BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ответ респондента
CREATE TABLE responses (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id          UUID NOT NULL REFERENCES participants(id),
    pair_presentation_id    UUID NOT NULL REFERENCES pair_presentations(id),
    choice                  VARCHAR(10) NOT NULL,  -- left | right | tie
    reason_codes            TEXT[],                 -- массив тегов
    confidence              INTEGER CHECK (confidence BETWEEN 1 AND 5),
    response_time_ms        INTEGER,
    replay_count            INTEGER NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(participant_id, pair_presentation_id)
);

-- Лог взаимодействий
CREATE TABLE interaction_logs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id          UUID NOT NULL REFERENCES participants(id),
    pair_presentation_id    UUID REFERENCES pair_presentations(id),
    event_type              VARCHAR(50) NOT NULL,
    event_ts                TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload_json            JSONB
);

CREATE INDEX idx_responses_participant ON responses(participant_id);
CREATE INDEX idx_responses_presentation ON responses(pair_presentation_id);
CREATE INDEX idx_interaction_participant ON interaction_logs(participant_id);
CREATE INDEX idx_pair_pres_participant ON pair_presentations(participant_id);
CREATE INDEX idx_source_items_study ON source_items(study_id);
CREATE INDEX idx_groups_study ON groups(study_id);

-- Удаляем старую таблицу votes (данные мигрированы в responses)
DROP TABLE IF EXISTS votes;
-- Удаляем старую таблицу comparisons
DROP TABLE IF EXISTS comparisons;
```

---

## Обновлённый REST API

### Public (респондент)

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/api/session/start` | Начать сессию (study_id, device, role, experience) → session_token + задания |
| `GET` | `/api/session/:token/next-task` | Следующее задание (pair_presentation с presigned URLs) |
| `POST` | `/api/task/:id/response` | Ответ: choice, reasons, confidence, response_time_ms, replay_count |
| `POST` | `/api/task/:id/event` | Лог события (page_loaded, replay_clicked, etc.) |
| `POST` | `/api/session/:token/complete` | Завершить сессию → completion code |

### Admin (JWT)

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/api/admin/login` | Аутентификация |
| **Studies** | | |
| `GET` | `/api/admin/studies` | Список исследований |
| `POST` | `/api/admin/studies` | Создать исследование |
| `PATCH` | `/api/admin/studies/:id` | Изменить статус (draft→active→paused→archived) |
| **Groups & Pairs** | | |
| `POST` | `/api/admin/studies/:id/groups` | Создать группу |
| `POST` | `/api/admin/studies/:id/import` | CSV-импорт пар (bulk upload) |
| `POST` | `/api/admin/assets/upload` | Загрузить видео-ассет |
| `GET` | `/api/admin/source-items` | Список пар с фильтрами |
| **Analytics** | | |
| `GET` | `/api/admin/analytics/overview` | Сводка: win rate, tie rate, по эффектам/группам |
| `GET` | `/api/admin/analytics/study/:id` | Детали по исследованию |
| `GET` | `/api/admin/analytics/qc` | QC отчёт: fast responses, straight-lining |
| **Export** | | |
| `GET` | `/api/admin/export/csv` | CSV-экспорт (1 строка = 1 ответ, 14 полей) |
| `GET` | `/api/admin/export/json` | JSON-экспорт |

---

## Страницы фронтенда (обновлённый flow)

```
Welcome → Instructions → Practice (2-3) → Tasks (15-25) → [Break] → Tasks → Completion
```

| # | Страница | Статус |
|---|---|---|
| 1 | **WelcomePage** — title, duration, consent, start | 🆕 |
| 2 | **InstructionsPage** — критерии, tie, replay, shortcuts | 🆕 |
| 3 | **PracticePage** — 2-3 тренировочных пары (не считаются) | 🆕 |
| 4 | **TaskPage** — два синхр. видео, A/B/Tie, reasons, confidence, progress bar, keyboard shortcuts | 🔄 (полная переработка VotingPage) |
| 5 | **BreakPage** — пауза после N заданий | 🆕 |
| 6 | **CompletionPage** — thank you, completion code | 🆕 |
| 7 | **LoginPage** — без изменений | ✅ |
| 8 | **AdminStudiesPage** — CRUD исследований | 🆕 |
| 9 | **AdminPairsPage** — CSV-импорт, список пар, QC-метки | 🔄 (переработка AdminComparisons) |
| 10 | **AdminAnalyticsPage** — win rate, tie rate, per-effect, QC, export | 🔄 (существенное расширение) |

---

## План по агентам

### 🔧 Агент 1 — Инфраструктура (доработка)

**Зависимости:** нет  
**Объём работ:** малый (2-3 файла)

| # | Задача | Действие |
|---|---|---|
| 1 | Миграция `002_study_schema.up.sql` | 🆕 Новый файл |
| 2 | Обновить `.env.example` (новые переменные если нужны) | 🔄 Минимально |
| 3 | Обновить seed-скрипт (создать тестовое study + группы) | 🔄 |

---

### ⚙️ Агент 2 — Backend (Go Gin) — основная работа

**Зависимости:** Агент 1  
**Объём работ:** большой

| # | Задача | Действие | Файлы |
|---|---|---|---|
| **Модели** | | | |
| 1 | Study, Group, SourceItem | 🆕 | `model/study.go`, `model/group.go`, `model/source_item.go` |
| 2 | VideoAsset (расширить Video) | 🔄 | `model/video.go` |
| 3 | Participant | 🆕 | `model/participant.go` |
| 4 | PairPresentation | 🆕 | `model/pair_presentation.go` |
| 5 | Response (вместо Vote) | 🔄 Переписать | `model/response.go` (удалить `vote.go`) |
| 6 | InteractionLog | 🆕 | `model/interaction_log.go` |
| **Репозитории** | | | |
| 7 | StudyRepo, GroupRepo | 🆕 | `repository/study.go`, `repository/group.go` |
| 8 | SourceItemRepo | 🆕 | `repository/source_item.go` |
| 9 | ParticipantRepo | 🆕 | `repository/participant.go` |
| 10 | PairPresentationRepo | 🆕 | `repository/pair_presentation.go` |
| 11 | ResponseRepo (вместо VoteRepo) | 🔄 Переписать | `repository/response.go` |
| 12 | InteractionLogRepo | 🆕 | `repository/interaction_log.go` |
| **Сервисы** | | | |
| 13 | SessionService — start, assign tasks, complete | 🆕 | `service/session.go` |
| 14 | AssignmentService — balanced sampling, randomization L/R | 🆕 | `service/assignment.go` |
| 15 | QCService — fast response detection, straight-lining, attention check validation | 🆕 | `service/qc.go` |
| 16 | ExportService — CSV/JSON export | 🆕 | `service/export.go` |
| 17 | AnalyticsService — win rate, tie rate, per-effect, per-group | 🔄 Расширить | `service/analytics.go` |
| 18 | ComparisonService → StudyService (импорт пар, группы) | 🔄 Переписать | `service/study.go` |
| **Handlers** | | | |
| 19 | SessionHandler — session/start, next-task, complete | 🆕 | `handler/session.go` |
| 20 | TaskHandler — response, event | 🆕 | `handler/task.go` |
| 21 | AdminHandler — studies CRUD, import, export, QC | 🔄 Переписать | `handler/admin.go` |
| 22 | Удалить `handler/comparison.go`, `handler/vote.go` | 🗑️ | |
| **Прочее** | | | |
| 23 | Обновить `main.go` — новые роуты, DI | 🔄 | `cmd/server/main.go` |
| 24 | CORS, rate limiting, CSRF | 🔄 | middleware |
| 25 | Unit-тесты | 🆕/🔄 | `*_test.go` |
| 26 | Integration/e2e тесты | 🆕 | `tests/` |
| 27 | golangci-lint | 🔄 | `.golangci.yml` |

---

### 🎨 Агент 3 — Frontend (React + Vite)

**Зависимости:** Агент 2  
**Объём работ:** большой (почти полная переработка страниц)

| # | Задача | Действие | Файлы |
|---|---|---|---|
| **Основной flow** | | | |
| 1 | API client — новые endpoints (session, task, event) | 🔄 Переписать | `api/client.js` |
| 2 | Session state management (context/store) | 🆕 | `context/SessionContext.jsx` |
| 3 | **WelcomePage** — title, estimated time, consent, start | 🆕 | `pages/WelcomePage.jsx` |
| 4 | **InstructionsPage** — критерии оценки, tie, replay | 🆕 | `pages/InstructionsPage.jsx` |
| 5 | **PracticePage** — тренировочные пары с подсказками | 🆕 | `pages/PracticePage.jsx` |
| 6 | **TaskPage** — синхр. видео, A/B/Tie, reasons, confidence, progress, keyboard | 🔄 Полная переработка | `pages/TaskPage.jsx` |
| 7 | **BreakPage** — пауза, мотивация | 🆕 | `pages/BreakPage.jsx` |
| 8 | **CompletionPage** — thank you, code | 🆕 | `pages/CompletionPage.jsx` |
| **Видеоплеер** | | | |
| 9 | SyncVideoPlayer — preload обоих, synchronization, replay both, keyboard (R) | 🔄 Полная переработка | `components/SyncVideoPlayer.jsx` |
| 10 | Preload next pair | 🆕 | логика в TaskPage |
| **Response panel** | | | |
| 11 | ChoicePanel — A/B/Tie кнопки + keyboard (1/2/0) | 🆕 | `components/ChoicePanel.jsx` |
| 12 | ReasonsSelector — до 2 тегов из 7 вариантов | 🆕 | `components/ReasonsSelector.jsx` |
| 13 | ConfidenceRating — 1-5 шкала | 🆕 | `components/ConfidenceRating.jsx` |
| 14 | ProgressBar — "Comparison 7 of 20" | 🆕 | `components/ProgressBar.jsx` |
| **Admin** | | | |
| 15 | AdminStudiesPage — CRUD исследований | 🆕 | `pages/AdminStudiesPage.jsx` |
| 16 | AdminPairsPage — CSV-импорт, список, QC-метки | 🔄 Переработка | `pages/AdminPairsPage.jsx` |
| 17 | AdminAnalyticsPage — win rate, tie rate, per-effect, QC, CSV export | 🔄 Расширить | `pages/AdminAnalyticsPage.jsx` |
| **UI/UX** | | | |
| 18 | Темы: нейтральный тёмный дизайн (убрать градиенты, "flashy" элементы) | 🔄 | `index.css` |
| 19 | Accessibility: focus states, readable fonts, large buttons | 🔄 | `index.css` |
| 20 | Mobile warning / desktop-first enforcement | 🆕 | |
| 21 | Routing update | 🔄 | `App.jsx` |
| 22 | Удалить `VotingPage.jsx`, `AdminComparisons.jsx` | 🗑️ | |

---

### 🧪 Агент 4 — QA (браузерная верификация)

**Зависимости:** Агенты 1-3  

| # | Сценарий | Что проверяем |
|---|---|---|
| 1 | Welcome → Instructions → Practice | Весь onboarding flow |
| 2 | Синхронизация видео | Оба стартуют вместе, replay обоих |
| 3 | A/B/Tie + reasons + confidence | Полная структура ответа сохраняется |
| 4 | Keyboard shortcuts (1/2/0/R/N) | Работают корректно |
| 5 | Progress bar + break page | Появляется после N заданий |
| 6 | Completion page + code | Показывается после последнего задания |
| 7 | Attention check (identical pair) | Не leak-ают что это проверка |
| 8 | Admin: создать study + CSV import | Пары создаются, видео загружены |
| 9 | Admin: analytics — win rate, tie rate | Данные корректны |
| 10 | Admin: CSV/JSON export | Файл содержит 14 полей на ответ |
| 11 | Method identity leakage | URL не содержат "baseline"/"candidate" |
| 12 | Network tab | Видео из S3, не через бэкенд |

---

## Verification Plan

### Automated
- `golangci-lint run ./...`
- `go test ./...` (unit)
- `go test -tags=integration ./tests/...` (e2e с testcontainers)
- `npm run build` (frontend)

### Manual (Агент 4)
- Полный сценарий из таблицы выше
