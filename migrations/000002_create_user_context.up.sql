CREATE TABLE Context (
    ID SERIAL PRIMARY KEY,
    goals TEXT[],
    recent5 TEXT[],
    progress JSONB
);