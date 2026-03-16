# Video Comparison Service

Платформа контролируемого парного сравнения видео (baseline vs candidate) для исследований эффектов flooding/explosion. Участники оценивают пары видео, администраторы управляют исследованиями и анализируют результаты.

## Стек

| Слой | Технология |
|---|---|
| Backend | Go 1.25 + Gin |
| Frontend | React 19 + Vite |
| База данных | PostgreSQL 16 |
| Хранилище видео | MinIO (S3-совместимое) |
| Контейнеры | Docker Compose |

---

## Быстрый старт

### 1. Клонировать и настроить окружение

```bash
git clone <repo-url>
cd comp-video-service
cp .env.example .env
```

Обязательно измените в `.env`:
```
JWT_SECRET=ваш_случайный_секрет_минимум_32_символа
```

По умолчанию локальный администратор создаётся автоматически:
```
DEFAULT_ADMIN_USERNAME=admin
DEFAULT_ADMIN_PASSWORD=admin_123_videogen
```

### 2. Запустить все сервисы

```bash
docker compose up -d --build
```

Сервисы поднимаются в порядке: PostgreSQL → MinIO → Backend → Frontend.
Миграции применяются автоматически при старте backend.
Администратор `admin` / `admin_123_videogen` создаётся или обновляется автоматически при старте backend.

### 3. Открыть в браузере

| URL | Назначение |
|---|---|
| http://localhost:5173 | Интерфейс участника |
| http://localhost:5173/admin/login | Панель администратора |
| http://localhost:9001 | MinIO Console (minioadmin / minioadmin) |

---

## Архитектура

```
browser
  ├── :5173  →  nginx (frontend SPA)
  │               └── /api/* proxy → backend:8080
  └── :9000  →  MinIO (прямые GET видео, публичный бакет)

backend:8080
  ├── /api/session/*   — участник (без авторизации)
  ├── /api/task/*      — участник (без авторизации)
  └── /api/admin/*     — администратор (JWT + CSRF)

postgres:5432  ←  backend
minio:9000     ←  backend (upload), browser (download)
```

---

## Workflow администратора

### 1. Создать исследование
`/admin/studies` → кнопка **+ Создать**

Поля:
- Название, тип эффекта (`flooding` / `explosion` / `mixed`)
- Количество заданий на участника
- Текст инструкций
- Опции: Tie (равны), Причины выбора, Уверенность

### 2. Создать группы
`/admin/pairs` → выбрать исследование → секция **Группы** → **+ Группа**

Группа = категория пар (например, «сцена 1», «городская локация»).
Скопируйте UUID группы — он нужен для CSV.

### 3. Импортировать пары через CSV

Формат файла:
```csv
group_id,source_image_id,pair_code,difficulty,is_attention_check,notes,baseline_s3_key,candidate_s3_key
550e8400-...,img_001,flood_001,easy,false,описание,,
550e8400-...,img_002,flood_002,medium,true,контрольная пара,,
```

| Колонка | Обязательная | Описание |
|---|---|---|
| `group_id` | **да** | UUID группы |
| `source_image_id` | нет | ваш внутренний ID изображения |
| `pair_code` | нет | читаемый код пары |
| `difficulty` | нет | `easy` / `medium` / `hard` |
| `is_attention_check` | нет | `true` / `false` |
| `notes` | нет | комментарий |
| `baseline_s3_key` | нет | S3-ключ уже загруженного видео |
| `candidate_s3_key` | нет | S3-ключ уже загруженного видео |

### 4. Загрузить видео-ассеты
`/admin/pairs` → секция **Загрузить видео-ассет**

Для каждой пары нужно загрузить **два** видео: `baseline` и `candidate`.
Задание назначается участнику только если оба ассета присутствуют.

### 5. Активировать исследование
`/admin/studies` → кнопка **→ Активно**

Только активные исследования доступны участникам.

### 6. Поделиться ссылкой
`/admin/studies` → кнопка **🔗 Ссылка** — копирует URL вида:
```
http://localhost:5173/?study_id=<uuid>
```

---

## Workflow участника

```
Приветствие → Инструкции → Практика → Задания → Пауза → Завершение
```

1. Открыть ссылку исследования
2. Заполнить роль и опыт
3. Ознакомиться с инструкциями
4. Выполнить тренировочные пары
5. Оценить видеопары: выбрать **A** / **B** / **Равны**, указать причину и уверенность
6. Получить код завершения

### Горячие клавиши на странице задания
| Клавиша | Действие |
|---|---|
| `1` | Выбрать левое видео (A) |
| `2` | Выбрать правое видео (B) |
| `0` | Равны (Tie) |
| `R` | Переиграть оба видео |
| `N` | Следующее задание (после выбора) |

---

## REST API

### Публичные эндпоинты (участник)

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/api/session/start` | Начать сессию → `session_token` + первое задание |
| `GET` | `/api/session/:token/next-task` | Следующее задание (204 если заданий нет) |
| `POST` | `/api/session/:token/complete` | Завершить → `completion_code` |
| `POST` | `/api/task/:id/response` | Отправить ответ |
| `POST` | `/api/task/:id/event` | Лог события (replay, etc.) |

### Административные эндпоинты (JWT + CSRF)

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/api/admin/login` | Авторизация → JWT + CSRF токен |
| `GET` | `/api/admin/studies` | Список исследований |
| `POST` | `/api/admin/studies` | Создать исследование |
| `PATCH` | `/api/admin/studies/:id` | Изменить статус |
| `GET` | `/api/admin/studies/:id/groups` | Список групп исследования |
| `POST` | `/api/admin/studies/:id/groups` | Создать группу |
| `POST` | `/api/admin/studies/:id/import` | CSV-импорт пар |
| `POST` | `/api/admin/assets/upload` | Загрузить видео-ассет (MP4) |
| `GET` | `/api/admin/source-items` | Список пар (фильтры: study_id, group_id) |
| `GET` | `/api/admin/analytics/overview` | Сводная аналитика |
| `GET` | `/api/admin/analytics/study/:id` | Детали по исследованию |
| `GET` | `/api/admin/analytics/qc` | QC-отчёт участников |
| `GET` | `/api/admin/export/csv` | Экспорт ответов в CSV |
| `GET` | `/api/admin/export/json` | Экспорт ответов в JSON |

---

## Структура проекта

```
comp-video-service/
├── backend/
│   ├── cmd/
│   │   ├── server/main.go        # Точка входа, роутер, DI
│   │   └── seed/main.go          # Скрипт создания admin
│   ├── internal/
│   │   ├── config/               # Загрузка env-переменных
│   │   ├── handler/              # HTTP-обработчики (Gin)
│   │   ├── middleware/           # JWT, CSRF, rate limit
│   │   ├── model/                # Go-структуры (модели данных)
│   │   ├── repository/           # Запросы к PostgreSQL (pgx/v5)
│   │   ├── service/              # Бизнес-логика
│   │   └── storage/              # S3/MinIO клиент
│   └── migrations/
│       ├── 001_init.up.sql       # Базовые таблицы
│       └── 002_study_schema.up.sql # Схема исследований
├── frontend/
│   └── src/
│       ├── api/client.js         # HTTP-клиент (fetch + JWT/CSRF)
│       ├── context/              # SessionContext, ToastContext
│       ├── components/           # SyncVideoPlayer, ChoicePanel, etc.
│       └── pages/                # Страницы участника и администратора
├── docker-compose.yml
├── .env.example
└── sample_pairs.csv              # Пример CSV для импорта пар
```

---

## Переменные окружения

| Переменная | Пример | Описание |
|---|---|---|
| `POSTGRES_DB` | `compvideo` | Имя БД |
| `POSTGRES_USER` | `cvs` | Пользователь БД |
| `POSTGRES_PASSWORD` | `cvs_secret` | Пароль БД |
| `MINIO_ROOT_USER` | `minioadmin` | Логин MinIO |
| `MINIO_ROOT_PASSWORD` | `minioadmin` | Пароль MinIO |
| `S3_BUCKET` | `videos` | Имя бакета |
| `S3_PUBLIC_URL` | `http://localhost:9000` | Публичный URL MinIO (видит браузер) |
| `BACKEND_PORT` | `8080` | Порт backend |
| `JWT_SECRET` | — | Секрет для JWT (обязательно сменить!) |
| `DEFAULT_ADMIN_USERNAME` | `admin` | Логин администратора для локального запуска |
| `DEFAULT_ADMIN_PASSWORD` | `admin_123_videogen` | Пароль администратора для локального запуска |
| `CORS_ORIGINS` | `http://localhost:5173` | Разрешённые origins |

---

## Разработка

### Backend (локально без Docker)

```bash
cd backend
go mod download
go run ./cmd/server
```

### Frontend (локально без Docker)

```bash
cd frontend
npm install
npm run dev   # http://localhost:5173
```

### Тесты

```bash
# Backend unit-тесты
cd backend && go test ./...

# Линтер
cd backend && golangci-lint run ./...

# Frontend линтер
cd frontend && npm run lint

# Frontend сборка
cd frontend && npm run build
```

### Применить миграции вручную

```bash
docker compose exec postgres psql -U cvs -d compvideo -f /docker-entrypoint-initdb.d/002_study_schema.up.sql
```
