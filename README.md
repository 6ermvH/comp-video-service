# Video Comparison Service

![Go version](https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white)
![Backend CI](https://github.com/6ermvH/comp-video-service/actions/workflows/backend.yml/badge.svg)
![Frontend CI](https://github.com/6ermvH/comp-video-service/actions/workflows/frontend.yml/badge.svg)
![API Contract](https://github.com/6ermvH/comp-video-service/actions/workflows/api-contract.yml/badge.svg)

Платформа для контролируемых перцептивных исследований методом попарного сравнения видео (baseline vs candidate). Участники оценивают пары видео, выбирая лучший вариант и указывая причину; администраторы управляют исследованиями и анализируют результаты.

## Стек технологий

| Слой | Технология |
|---|---|
| Backend | Go 1.25 + Gin |
| Frontend | React 19 + Vite |
| База данных | PostgreSQL 16 |
| Хранилище видео | MinIO (dev) / Timeweb S3 (prod) |
| Контейнеры | Docker Compose |
| Деплой | Timeweb Cloud App Platform |
| CI | GitHub Actions |

---

## Возможности

**Панель администратора**
- Создание и управление исследованиями, группами пар и видеоматериалами
- Импорт ZIP-архива с видео через чанкованную загрузку (до 1 ГБ)
- Назначение пар участникам с рандомизацией порядка
- QC-флаги для выявления подозрительных участников
- Аналитический дашборд с агрегированной статистикой
- Экспорт результатов в CSV с вычисляемыми полями (`candidate_chosen`, `choice` как candidate/baseline/tie, `custom_reason`)

**Сессия участника**
- Рандомизированный порядок заданий
- Сравнение двух видео с выбором лучшего (A / B / Tie)
- Коды причин, рейтинг уверенности, произвольный комментарий
- Практические пары перед основными заданиями
- Клавиатурные сокращения для быстрой оценки

**Инфраструктура**
- Presigned S3 URL для безопасной доставки видео
- Автоматическое применение миграций при старте
- Rate limiting (300 запросов/минуту)
- JWT + CSRF для защиты admin API

---

## Быстрый старт (локально)

### 1. Клонировать и настроить

```bash
git clone <repo-url>
cd comp-video-service
cp .env.example .env
```

Обязательно задайте надёжный JWT-секрет:

```
JWT_SECRET=your_random_secret_at_least_32_chars
```

### 2. Запустить сервисы

```bash
docker compose up -d --build
```

Порядок запуска: PostgreSQL → MinIO → Backend → Frontend.
Миграции применяются автоматически при старте бэкенда.

### 3. Создать первого администратора

```bash
SEED_USERNAME=admin SEED_PASSWORD=yourpassword docker compose run --rm seed
```

Или раскомментируйте сервис `seed` в `docker-compose.yml` и задайте переменные в `.env`.

### 4. Открыть в браузере

| URL | Назначение |
|---|---|
| http://localhost:5173 | Интерфейс участника |
| http://localhost:5173/admin/login | Панель администратора |
| http://localhost:9001 | MinIO Console |
| http://localhost:8080/swagger/index.html | Swagger UI |

---

## Переменные окружения

Локальная разработка использует `.env` в корне проекта. Для продакшна переменные задаются в Timeweb Cloud.

| Переменная | Пример (dev) | Описание |
|---|---|---|
| `PORT` | `8080` | Порт бэкенда |
| `DATABASE_URL` | `postgres://cvs:cvs_secret@postgres:5432/compvideo` | Строка подключения к PostgreSQL |
| `JWT_SECRET` | — | Секрет JWT (**обязательно сменить**) |
| `S3_ENDPOINT` | `http://minio:9000` | URL S3-совместимого хранилища |
| `S3_REGION` | `ru-1` | Регион (для Timeweb S3) |
| `S3_BUCKET` | `videos` | Имя бакета |
| `S3_ACCESS_KEY` | `minioadmin` | Ключ доступа S3 |
| `S3_SECRET_KEY` | `minioadmin` | Секретный ключ S3 |
| `S3_PUBLIC_URL` | `http://localhost:9000` | Публичный URL хранилища (видит браузер) |
| `S3_USE_PATH_STYLE` | `true` | Path-style доступ (нужен для MinIO и Timeweb S3) |
| `S3_USE_SSL` | `false` | TLS для S3 соединения |
| `CORS_ORIGINS` | `http://localhost:5173` | Разрешённые источники CORS |
| `MIGRATIONS_PATH` | `file:///migrations` | Путь к файлам миграций |
| `SEED_USERNAME` | `admin` | Логин создаваемого администратора |
| `SEED_PASSWORD` | — | Пароль администратора (**обязателен**) |
| `MINIO_ROOT_USER` | `minioadmin` | Логин MinIO (только для docker-compose) |
| `MINIO_ROOT_PASSWORD` | `minioadmin` | Пароль MinIO (только для docker-compose) |

---

## Архитектура

```
browser
  ├── :5173  →  Vite dev server (frontend SPA)
  │               └── /api/* proxy → backend:8080
  └── :9000  →  MinIO (прямой GET видео через presigned URL)

backend:8080
  ├── /api/session/*   — участник (без auth)
  ├── /api/task/*      — участник (без auth)
  └── /api/admin/*     — admin (JWT + CSRF)

postgres:5432  ←  backend (внутренняя Docker-сеть)
minio:9000     ←  backend (upload presigned), browser (download presigned)
```

---

## Рабочий процесс администратора

### 1. Создать исследование
`/admin/studies` → **+ Создать**

Параметры: название, тип эффекта (`flooding` / `explosion` / `mixed`), количество заданий на участника, инструкции, опции (разрешить ничью, коды причин, уверенность).

### 2. Создать группы
`/admin/pairs` → выбрать исследование → раздел **Группы** → **+ Группа**

Группа — категория пар (например, «сцена 1», «городская локация»).

### 3. Импорт видео через ZIP-архив
`/admin/studies` → **Импорт архива**

Загрузите ZIP (до 1 ГБ) с видеофайлами. Используется чанкованная загрузка, которая автоматически разбивает файл на части.

Альтернатива — CSV-импорт пар с уже загруженными S3-ключами:

```csv
group_id,source_image_id,pair_code,difficulty,is_attention_check,notes,baseline_s3_key,candidate_s3_key
550e8400-...,img_001,flood_001,easy,false,описание,,
```

### 4. Загрузить видеоматериалы
`/admin/pairs` → **Загрузить видео**

Каждая пара требует два видео: `baseline` и `candidate`. Пара попадает к участникам только когда оба видео загружены.

### 5. Активировать исследование
`/admin/studies` → **→ Активное**

Только активные исследования доступны участникам.

### 6. Поделиться ссылкой
`/admin/studies` → кнопка **Ссылка** — копирует URL вида:
```
http://localhost:5173/?study_id=<uuid>
```

---

## Рабочий процесс участника

```
Приветствие → Инструкции → Практика → Задания → Завершение
```

1. Открыть ссылку на исследование
2. Заполнить роль и опыт
3. Прочитать инструкции
4. Выполнить практические пары
5. Оценить видеопары: выбрать **A** / **B** / **Ничья**, указать причину и уверенность
6. Получить код завершения

### Клавиатурные сокращения

| Клавиша | Действие |
|---|---|
| `1` | Выбрать левое видео (A) |
| `2` | Выбрать правое видео (B) |
| `0` | Ничья |
| `R` | Воспроизвести оба видео заново |
| `N` | Следующее задание (после выбора) |

---

## REST API

Интерактивная документация: **`/swagger/index.html`** (Swagger UI)

Спецификация OpenAPI: `backend/docs/swagger.yaml`

### Публичные эндпоинты (участник)

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/api/session/start` | Начать сессию → `session_token` + первое задание |
| `GET` | `/api/session/:token/next-task` | Следующее задание (204 если заданий нет) |
| `POST` | `/api/session/:token/complete` | Завершить → `completion_code` |
| `POST` | `/api/task/:id/response` | Отправить ответ |
| `POST` | `/api/task/:id/event` | Логировать событие (повтор и т.д.) |

### Admin эндпоинты (JWT + CSRF)

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/api/admin/login` | Войти → JWT + CSRF-токен |
| `GET` | `/api/admin/studies` | Список исследований |
| `POST` | `/api/admin/studies` | Создать исследование |
| `PATCH` | `/api/admin/studies/:id` | Обновить исследование/статус |
| `DELETE` | `/api/admin/studies/:id` | Удалить исследование (только draft/archived) |
| `POST` | `/api/admin/studies/import-archive` | Импорт ZIP-архива с видео |
| `GET` | `/api/admin/studies/:id/groups` | Список групп |
| `POST` | `/api/admin/studies/:id/groups` | Создать группу |
| `POST` | `/api/admin/studies/:id/pairs` | Создать пару |
| `GET` | `/api/admin/source-items` | Список пар (фильтры: study_id, group_id) |
| `PATCH` | `/api/admin/source-items/:id` | Обновить пару |
| `DELETE` | `/api/admin/source-items/:id` | Удалить пару |
| `POST` | `/api/admin/assets/upload` | Загрузить видеоматериал (MP4) |
| `GET` | `/api/admin/assets` | Список материалов |
| `GET` | `/api/admin/assets/free` | Материалы без привязки к паре |
| `GET` | `/api/admin/assets/:id/url` | Presigned URL для просмотра |
| `DELETE` | `/api/admin/assets/:id` | Удалить материал |
| `POST` | `/api/admin/uploads/init` | Инициировать чанкованную загрузку |
| `POST` | `/api/admin/uploads/:id/chunks/:index` | Загрузить чанк |
| `POST` | `/api/admin/uploads/:id/complete` | Завершить чанкованную загрузку |
| `DELETE` | `/api/admin/uploads/:id` | Отменить чанкованную загрузку |
| `GET` | `/api/admin/analytics/overview` | Сводная аналитика |
| `GET` | `/api/admin/analytics/study/:id` | Аналитика по исследованию |
| `GET` | `/api/admin/analytics/study/:id/pairs` | Аналитика по парам |
| `GET` | `/api/admin/analytics/qc` | QC-отчёт по участникам |
| `GET` | `/api/admin/export/csv` | Экспорт всех ответов в CSV |
| `GET` | `/api/admin/export/study/:id/csv` | Экспорт ответов исследования в CSV |

---

## Деплой на Timeweb Cloud

Проект настроен для деплоя на **Timeweb Cloud App Platform**. Корневой `Dockerfile` собирает бэкенд из поддиректории `backend/`.

При деплое необходимо задать все переменные окружения через панель Timeweb, используя реальные значения для Timeweb S3 (`S3_ENDPOINT=https://s3.timeweb.cloud`, `S3_REGION=ru-1`).

Фронтенд деплоится отдельно (статика через CDN или отдельный сервис).

---

## Структура проекта

```
comp-video-service/
├── backend/
│   ├── cmd/
│   │   ├── server/main.go        # Точка входа, роутер, DI
│   │   └── seed/main.go          # CLI для создания администратора
│   ├── internal/
│   │   ├── config/               # Конфигурация из env
│   │   ├── handler/              # Gin HTTP-хендлеры
│   │   ├── middleware/           # JWT, CSRF, rate limiting
│   │   ├── model/                # Доменные структуры
│   │   ├── repository/           # PostgreSQL-запросы (pgx/v5)
│   │   ├── service/              # Бизнес-логика
│   │   └── storage/              # S3/MinIO клиент
│   ├── migrations/               # SQL-миграции (golang-migrate)
│   ├── tests/                    # Интеграционные тесты (build tag: integration)
│   └── docs/                     # OpenAPI спецификация (swagger.yaml, swagger.json)
├── frontend/
│   └── src/
│       ├── api/client.js         # Все HTTP-вызовы — единый источник истины
│       ├── context/              # SessionContext, ToastContext
│       ├── components/           # SyncVideoPlayer, ChoicePanel и др.
│       └── pages/                # Страницы участника и администратора
├── .github/workflows/
│   ├── backend.yml               # Lint, unit tests, coverage ≥70%, build
│   ├── frontend.yml              # Lint, build
│   └── api-contract.yml          # Проверка drift OpenAPI + контракт фронтенда
├── Dockerfile                    # Root Dockerfile для Timeweb App Platform
├── docker-compose.yml
├── .env.example
└── sample_pairs.csv              # Пример CSV для импорта пар
```

---

## Разработка

### Backend (без Docker)

```bash
cd backend
go mod download
go run ./cmd/server
```

### Frontend (без Docker)

```bash
cd frontend
npm install
npm run dev   # http://localhost:5173
```

### Тесты

```bash
# Unit-тесты бэкенда
cd backend && go test ./... -race -count=1

# Интеграционные тесты (требуют Docker)
cd backend && go test -tags integration ./tests/... -v

# Линтер бэкенда
cd backend && golangci-lint run ./...

# Линтер фронтенда
cd frontend && npm run lint

# Сборка фронтенда
cd frontend && npm run build

# Валидация API-контракта
cd frontend && npm run validate:api
```

### Обновить OpenAPI спецификацию

```bash
cd backend
swag init -g cmd/server/main.go -o docs --outputTypes yaml,json --quiet
```

Закоммитить обновлённые `docs/swagger.yaml` и `docs/swagger.json`.

---

## CI

| Воркфлоу | Триггеры | Что проверяет |
|---|---|---|
| `backend.yml` | `backend/**` | lint, unit tests, coverage ≥70%, сборка бинарника, интеграционные тесты |
| `frontend.yml` | `frontend/**` | lint, production build |
| `api-contract.yml` | `backend/**`, `frontend/src/api/**` | spec не дрейфует, все API-вызовы фронтенда существуют в спецификации |

---

