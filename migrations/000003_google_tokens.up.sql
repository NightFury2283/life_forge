CREATE TABLE lifeforge.google_tokens (
    user_id INTEGER PRIMARY KEY REFERENCES lifeforge.users(id) ON DELETE CASCADE,
    encrypted_token BYTEA NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
