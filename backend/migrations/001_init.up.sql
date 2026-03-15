-- ============================================================
-- 001_init.up.sql  — initial schema for comp-video-service
-- ============================================================

-- Enable pgcrypto for gen_random_uuid() if not already enabled
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ----------------------------------------------------------
-- videos
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS videos (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    s3_key      VARCHAR(512) NOT NULL UNIQUE,
    duration_ms INTEGER,
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',  -- active | archived
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ----------------------------------------------------------
-- comparisons
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS comparisons (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(255),
    video_a_id  UUID        NOT NULL REFERENCES videos(id),
    video_b_id  UUID        NOT NULL REFERENCES videos(id),
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_comparison_pair UNIQUE (video_a_id, video_b_id)
);

-- ----------------------------------------------------------
-- votes
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS votes (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    comparison_id   UUID        NOT NULL REFERENCES comparisons(id),
    chosen_video_id UUID        NOT NULL REFERENCES videos(id),
    session_id      VARCHAR(64) NOT NULL,
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- prevent duplicate vote from same session for same comparison
    CONSTRAINT uq_vote_session UNIQUE (comparison_id, session_id)
);

CREATE INDEX IF NOT EXISTS idx_votes_comparison ON votes(comparison_id);
CREATE INDEX IF NOT EXISTS idx_votes_chosen     ON votes(chosen_video_id);

-- ----------------------------------------------------------
-- admins
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS admins (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
