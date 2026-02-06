CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    is_event BOOLEAN DEFAULT TRUE,
    title VARCHAR(255) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE,
    duration_hours FLOAT,
    recurrence VARCHAR(50),
    description TEXT
);