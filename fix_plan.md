# Fix Plan — Backend (на основе логов docker-compose)

## Диагностика из логов

### Подтверждённые ошибки

| Endpoint | HTTP | Ошибка |
|---|---|---|
| `GET /api/admin/studies` | 500 | `scan study: can't scan into dest[5] (col: instructions_text): cannot scan NULL into *string` |
| `POST /api/admin/studies` | 500 | та же причина (RETURNING сразу же читает строку) |
| `GET /api/admin/analytics/overview` | 500 → исправлено само (миграция применена позже) |
| `GET /api/admin/analytics/qc` | 500 → исправлено само |
| `POST /api/session/start` | 400 | `invalid UUID length: 4` — тест с study_id="test", не баг кода |

**Корневая причина 500:** миграция `002_study_schema.up.sql` изначально не применялась (`docker-entrypoint-initdb.d` запускается только при первой инициализации тома). После ручного применения таблицы появились, но `GET /api/admin/studies` **по-прежнему возвращает 500** из-за NULL-скана.

---

## Баг #1 — CRITICAL: NULL scan в `studies.instructions_text`

**Файл:** `backend/internal/model/study.go:16`
**Причина:** поле `instructions_text` в БД — `TEXT` (nullable), Go-тип — `string` (non-pointer). pgx/v5 не умеет скандировать `NULL` в `string`.

### Затронутые эндпоинты
- `GET /api/admin/studies` — 500 при наличии хоть одной записи с `NULL instructions_text`
- `POST /api/admin/studies` — 500 при создании без `instructions_text` (RETURNING тоже читает поле)
- `PATCH /api/admin/studies/:id` — 500

### Фикс

**`model/study.go`**
```go
// было
InstructionsText string `json:"instructions_text,omitempty"`

// стало
InstructionsText *string `json:"instructions_text,omitempty"`
```

Репозиторий (`repository/study.go`) менять не нужно — `scanStudy` использует `&s.InstructionsText`, pgx корректно пишет NULL в `*string`.

Места использования поля в сервисах/хендлерах:
- `service/study.go` — если поле передаётся в шаблон, убедиться что nil-safe (`if s.InstructionsText != nil`)
- `handler/session.go` — `InstructionsText` передаётся в `SessionMeta` как `instructions_text`; обернуть разыменование

---

## Баг #2 — HIGH: NULL scan в остальных nullable строковых полях

Такая же проблема существует в четырёх других моделях — проявится при чтении записей с NULL значениями.

### `model/group.go:14` — `Description string`
БД: `description TEXT` (nullable). Проявится при `GET /api/admin/source-items` или любом запросе, возвращающем группы.

```go
// было
Description string `json:"description,omitempty"`
// стало
Description *string `json:"description,omitempty"`
```

### `model/participant.go:14–17,20` — DeviceType, Browser, Role, Experience, QualityFlag
Все nullable в БД; CREATE вставляет значения из реквеста, но если придёт `""` или поле не передано, БД запишет `""` (не NULL). Однако `quality_flag` имеет `DEFAULT 'ok'` — не проблема при создании. Тем не менее при прямых манипуляциях с БД или в тестах могут быть NULL.

Минимальный фикс — `QualityFlag` точно может быть NULL (колонка без NOT NULL):
```go
QualityFlag *string `json:"quality_flag,omitempty"`
```

Остальные поля (`DeviceType`, `Browser`, `Role`, `Experience`) — низкий риск, т.к. всегда вставляются из реквеста.

### `model/source_item.go:14–16,18` — SourceImageID, PairCode, Difficulty, Notes
Все nullable. Проявится при `GET /api/admin/source-items` если CSV-импорт оставил поля пустыми (NULL).

```go
SourceImageID *string `json:"source_image_id,omitempty"`
PairCode      *string `json:"pair_code,omitempty"`
Difficulty    *string `json:"difficulty,omitempty"`
Notes         *string `json:"notes,omitempty"`
```

### `model/video.go:21–23,29–30` — MethodType, Description, Codec, Checksum
Nullable в БД. Проявится при чтении видео без метаданных.

```go
MethodType  *string `json:"method_type,omitempty"`
Description *string `json:"description,omitempty"`
Codec       *string `json:"codec,omitempty"`
Checksum    *string `json:"checksum,omitempty"`
```

---

## Баг #3 — MEDIUM: `docker-compose.yml` — миграция не применяется автоматически

**Проблема:** `./backend/migrations:/docker-entrypoint-initdb.d` срабатывает **только при первом запуске** (пустой том). При `docker compose up` на уже инициализированном томе `002_study_schema.up.sql` молча игнорируется.

В логах это и произошло: сначала все 500, потом ручное применение.

### Фикс — добавить init-контейнер или migrate утилиту в backend Dockerfile

Вариант А — в `docker-compose.yml` добавить `depends_on` с хелсчеком и применять миграции в entrypoint `backend`:

```yaml
# backend service command/entrypoint
command: sh -c "
  until pg_isready -h postgres -U $POSTGRES_USER; do sleep 1; done &&
  psql $DATABASE_URL -f /migrations/001_init.up.sql &&
  psql $DATABASE_URL -f /migrations/002_study_schema.up.sql &&
  /app/server
"
```

Вариант Б — использовать [golang-migrate](https://github.com/golang-migrate/migrate) в `main.go`:
```go
m, _ := migrate.New("file:///migrations", cfg.DatabaseURL)
m.Up() // idempotent
```

**Рекомендуется Вариант Б** — migrate корректно отслеживает уже применённые версии.

---

## Баг #4 — LOW: `docker-compose.yml` — obsolete `version` атрибут

```
level=warning msg="...the attribute `version` is obsolete"
```

Просто удалить строку `version: "3.9"` (или аналог) из `docker-compose.yml`.

---

## Порядок исправлений

| # | Приоритет | Действие | Файл |
|---|---|---|---|
| 1 | CRITICAL | `InstructionsText string` → `*string` + проверить nil-использование | `model/study.go` |
| 2 | HIGH | `Description string` → `*string` в Group | `model/group.go` |
| 3 | HIGH | `SourceImageID`, `PairCode`, `Difficulty`, `Notes` → `*string` | `model/source_item.go` |
| 4 | HIGH | `MethodType`, `Description`, `Codec`, `Checksum` → `*string` | `model/video.go` |
| 5 | MEDIUM | `QualityFlag` → `*string` | `model/participant.go` |
| 6 | MEDIUM | Автоматическое применение миграций при старте backend | `cmd/server/main.go` или `docker-compose.yml` |
| 7 | LOW | Удалить `version:` из docker-compose | `docker-compose.yml` |

---

## Проверка после фикса

```bash
# 1. Пересобрать и поднять
docker compose build backend && docker compose up -d backend

# 2. Получить токен
TOKEN=$(curl -s -X POST http://localhost:5173/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r .token)

# 3. Проверить studies
curl -s http://localhost:5173/api/admin/studies \
  -H "Authorization: Bearer $TOKEN" | jq .

# 4. Создать study и убедиться что list работает
curl -s -X POST http://localhost:5173/api/admin/studies \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: $CSRF" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","effect_type":"flooding"}' | jq .

# 5. Убедиться что study без instructions_text читается корректно
curl -s http://localhost:5173/api/admin/studies \
  -H "Authorization: Bearer $TOKEN" | jq .
```
