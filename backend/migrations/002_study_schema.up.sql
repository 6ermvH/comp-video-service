-- ============================================================
-- 002_study_schema.up.sql
-- ============================================================

-- Исследование (flooding benchmark, explosion benchmark, etc.)
CREATE TABLE IF NOT EXISTS studies (
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
CREATE TABLE IF NOT EXISTS groups (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id            UUID NOT NULL REFERENCES studies(id),
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    priority            INTEGER NOT NULL DEFAULT 0,
    target_votes_per_pair INTEGER NOT NULL DEFAULT 10,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Исходный элемент (одно изображение → 2 видео)
CREATE TABLE IF NOT EXISTS source_items (
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
-- Rename existing table and add columns
ALTER TABLE videos RENAME TO video_assets;
ALTER TABLE video_assets ADD COLUMN IF NOT EXISTS source_item_id UUID REFERENCES source_items(id);
ALTER TABLE video_assets ADD COLUMN IF NOT EXISTS method_type VARCHAR(20); -- baseline | candidate
ALTER TABLE video_assets ADD COLUMN IF NOT EXISTS width INTEGER;
ALTER TABLE video_assets ADD COLUMN IF NOT EXISTS height INTEGER;
ALTER TABLE video_assets ADD COLUMN IF NOT EXISTS fps REAL;
ALTER TABLE video_assets ADD COLUMN IF NOT EXISTS codec VARCHAR(50);
ALTER TABLE video_assets ADD COLUMN IF NOT EXISTS checksum VARCHAR(128);

-- Участник (один респондент)
CREATE TABLE IF NOT EXISTS participants (
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
CREATE TABLE IF NOT EXISTS pair_presentations (
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
CREATE TABLE IF NOT EXISTS responses (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id          UUID NOT NULL REFERENCES participants(id),
    pair_presentation_id    UUID NOT NULL REFERENCES pair_presentations(id),
    choice                  VARCHAR(10) NOT NULL,  -- left | right | tie
    reason_codes            TEXT[],                 -- массив тегов
    confidence              INTEGER CHECK (confidence BETWEEN 1 AND 5),
    response_time_ms        INTEGER,
    replay_count            INTEGER NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_response_presentation UNIQUE(participant_id, pair_presentation_id)
);

-- Лог взаимодействий
CREATE TABLE IF NOT EXISTS interaction_logs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id          UUID NOT NULL REFERENCES participants(id),
    pair_presentation_id    UUID REFERENCES pair_presentations(id),
    event_type              VARCHAR(50) NOT NULL,
    event_ts                TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload_json            JSONB
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_responses_participant ON responses(participant_id);
CREATE INDEX IF NOT EXISTS idx_responses_presentation ON responses(pair_presentation_id);
CREATE INDEX IF NOT EXISTS idx_interaction_participant ON interaction_logs(participant_id);
CREATE INDEX IF NOT EXISTS idx_pair_pres_participant ON pair_presentations(participant_id);
CREATE INDEX IF NOT EXISTS idx_source_items_study ON source_items(study_id);
CREATE INDEX IF NOT EXISTS idx_groups_study ON groups(study_id);

-- Cleanup old tables (their data will be lost safely since we haven't launched yet)
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS comparisons;
