CREATE SCHEMA lifeforge

CREATE TABLE lifeforge.users (
    id SERIAL PRIMARY KEY,
    google_id TEXT UNIQUE NOT NULL,
    email TEXT,
    name TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE lifeforge.user_info (
    user_id INTEGER PRIMARY KEY REFERENCES lifeforge.users(id) ON DELETE CASCADE,
    goals TEXT[],
    progress_context JSONB,

    -- индекс дисциплины
    reliability_score FLOAT NOT NULL DEFAULT 75 CHECK(reliability_score BETWEEN 0 AND 100), 

    --саммари привычек для ИИ
    behavior_summary TEXT,

    --хронотип (лучшее время для работы)
    best_time_window VARCHAR(25)
);

CREATE TABLE lifeforge.user_progress (
    user_id INTEGER PRIMARY KEY REFERENCES lifeforge.users(id) ON DELETE CASCADE,
    level INTEGER NOT NULL DEFAULT 1 CHECK(level > 0),
    xp INTEGER NOT NULL DEFAULT 0 CHECK(xp >= 0),
    xp_for_next_level INTEGER NOT NULL DEFAULT 100,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    coins INTEGER NOT NULL DEFAULT 0 CHECK(coins>=0)
);

CREATE TABLE lifeforge.user_stats (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES lifeforge.users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    value INTEGER DEFAULT 0 CHECK(value>=0),
    UNIQUE(user_id, name)
);