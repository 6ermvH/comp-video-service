# Задачи для Агента 3 — Backend

## Задача 1 — CRITICAL: NULL scan — GET /api/admin/studies возвращает 500

**Файл:** `backend/internal/model/study.go`, строка 16

**Причина:** `instructions_text` в БД — `TEXT` (nullable), Go-тип — `string`. pgx/v5 не умеет скандировать NULL в non-pointer `string`.

**Фикс:**
```go
// было
InstructionsText string `json:"instructions_text,omitempty"`
// стало
InstructionsText *string `json:"instructions_text,omitempty"`
```

Репозиторий (`repository/study.go`) — менять не нужно, `scanStudy` через `&s.InstructionsText` теперь корректно получит `*string`.

Проверить использование поля в `service/session.go` (передаётся в `SessionMeta`) — при передаче разыменовать безопасно через `""` если nil.

---

## Задача 2 — HIGH: NULL scan в остальных моделях (проявится при наличии NULL в БД)

### `backend/internal/model/group.go:14`
```go
// было
Description string `json:"description,omitempty"`
// стало
Description *string `json:"description,omitempty"`
```

### `backend/internal/model/source_item.go` — строки 14–18
```go
// было
SourceImageID string `json:"source_image_id,omitempty"`
PairCode      string `json:"pair_code,omitempty"`
Difficulty    string `json:"difficulty,omitempty"`
Notes         string `json:"notes,omitempty"`
// стало — все четыре поля
SourceImageID *string `json:"source_image_id,omitempty"`
PairCode      *string `json:"pair_code,omitempty"`
Difficulty    *string `json:"difficulty,omitempty"`
Notes         *string `json:"notes,omitempty"`
```

### `backend/internal/model/video.go` — строки 21–23, 29–30
```go
// было
MethodType  string `json:"method_type,omitempty"`
Description string `json:"description"`
Codec       string `json:"codec,omitempty"`
Checksum    string `json:"checksum,omitempty"`
// стало
MethodType  *string `json:"method_type,omitempty"`
Description *string `json:"description,omitempty"`
Codec       *string `json:"codec,omitempty"`
Checksum    *string `json:"checksum,omitempty"`
```

### `backend/internal/model/participant.go:20`
```go
// было
QualityFlag string `json:"quality_flag"`
// стало
QualityFlag *string `json:"quality_flag,omitempty"`
```

**После изменений** — найти все места в сервисах и хендлерах где эти поля используются как `string` (сравнения, конкатенации) и обернуть nil-safe разыменованием.

---

## Задача 3 — MEDIUM: GET /api/admin/source-items возвращает `null` вместо `[]`

**Файл:** `backend/internal/repository/source_item.go`

При пустом результате `ListWithFilters` и `ListByStudy` возвращают `nil` (Go nil slice). JSON-маршалинг превращает nil slice в `null`, а не `[]`.

**Фикс в обоих методах** — инициализировать слайс:
```go
// было
var out []*model.SourceItem
// стало
out := make([]*model.SourceItem, 0)
```

Аналогично проверить `repository/pair_presentation.go`, `repository/response.go` — аналогичная потенциальная проблема.

---

## Задача 4 — MEDIUM: GET /session/:token/next-task возвращает 500

В логах: `GET "/api/session/0c942e53.../next-task" 500`

Нужно воспроизвести ошибку и изучить логи бэкенда с деталью ошибки:
```bash
docker compose logs backend 2>&1 | grep -A3 "next-task"
```
Скорее всего связано с NULL scan в моделях (Задача 1/2) или отсутствием видео-ассетов у пары. Исправить после Задач 1–2.

---

## Задача 5 — HIGH: Добавить endpoint GET /api/admin/studies/:id/groups

**Проблема:** на странице `/admin/pairs` нет способа получить список групп исследования с их UUID. Без UUID группы нельзя составить CSV для импорта пар — первая колонка CSV это `group_id`.

**Что нужно:**

### Репозиторий — `backend/internal/repository/group.go`
Добавить метод `ListByStudy`:
```go
func (r *GroupRepository) ListByStudy(ctx context.Context, studyID uuid.UUID) ([]*model.Group, error)
```
SQL: `SELECT id, study_id, name, description, priority, target_votes_per_pair, created_at FROM groups WHERE study_id = $1 ORDER BY priority ASC, created_at ASC`

Инициализировать результат через `make([]*model.Group, 0)`.

### Сервис — `backend/internal/service/study.go`
Добавить метод:
```go
func (s *StudyService) ListGroups(ctx context.Context, studyID uuid.UUID) ([]*model.Group, error) {
    return s.groupRepo.ListByStudy(ctx, studyID)
}
```

### Хендлер — `backend/internal/handler/admin.go`
Добавить метод `ListGroups`:
```go
// GET /api/admin/studies/:id/groups
func (h *AdminHandler) ListGroups(c *gin.Context) {
    studyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study id"})
        return
    }
    groups, err := h.studySvc.ListGroups(c.Request.Context(), studyID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"groups": groups})
}
```

### Роутер — `backend/cmd/server/main.go`
Добавить маршрут в adminGroup:
```go
adminGroup.GET("/studies/:id/groups", adminH.ListGroups)
```

---

## Задача 6 — CRITICAL: Видео не загружается — presigned URL возвращает 403

**Файл:** `backend/internal/storage/s3.go`, метод `PresignedURL`

**Причина:** подпись генерируется для внутреннего хоста `http://minio:9000` (S3_ENDPOINT), затем хост в URL подменяется на `http://localhost:9000` (S3_PUBLIC_URL). MinIO отклоняет запрос (403) потому что Host в подписи не совпадает с Host в запросе.

**Проверено:** прямой GET без подписи `http://localhost:9000/videos/{key}` возвращает 200 — бакет публичный (политика `download`).

**Фикс:** заменить presigning на прямую публичную URL:

```go
// backend/internal/storage/s3.go

// PresignedURL returns a public URL for the given key.
// The MinIO bucket has public download policy, so no signing is needed.
func (s *S3Client) PresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
    return fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucket, key), nil
}
```

Убрать неиспользуемое поле `presigner *s3.PresignClient` из структуры и удалить его инициализацию `s3.NewPresignClient(client)`.

Неиспользуемые импорты удалить: `"github.com/aws/aws-sdk-go-v2/service/s3"` (presign-методы).

---

## Задача 7 — CRITICAL: 500 после последнего задания вместо перехода к завершению

**Файл:** `backend/internal/handler/session.go`, строка 50

**Причина:** ошибка из репозитория обёрнута через `fmt.Errorf("scan pair presentation: %w", err)`, поэтому прямое сравнение `err == pgx.ErrNoRows` возвращает `false`. Хендлер уходит в ветку 500 вместо 204.

**Фикс:**
```go
// было
import "github.com/jackc/pgx/v5"
...
if err == pgx.ErrNoRows {

// стало
import (
    "errors"
    "github.com/jackc/pgx/v5"
)
...
if errors.Is(err, pgx.ErrNoRows) {
```

**Файл:** `backend/internal/service/session.go`, строка 107 — аналогичная проверка:
```go
// было
if err != nil && err != pgx.ErrNoRows {
// стало
if err != nil && !errors.Is(err, pgx.ErrNoRows) {
```
