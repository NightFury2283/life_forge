CREATE TABLE lifeforge.completed_tasks (
    user_id INTEGER NOT NULL REFERENCES lifeforge.users(id) ON DELETE CASCADE,
    calendar_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    title TEXT NOT NULL,
    xp INTEGER NOT NULL DEFAULT 10,
    coins INTEGER NOT NULL DEFAULT 2,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(user_id,calendar_id,event_id)
);
CREATE INDEX completed_tasks_user_time ON lifeforge.completed_tasks(user_id,completed_at);
