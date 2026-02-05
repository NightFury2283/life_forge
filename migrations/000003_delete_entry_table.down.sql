CREATE TABLE journal_entries (
    ID SERIAL PRIMARY KEY,
    entry_text TEXT NOT NULL,
    mood_score INT,
    created_at TIMESTAMP DEFAULT NOW()
);