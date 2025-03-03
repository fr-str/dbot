-- +goose up
CREATE TABLE IF NOT EXISTS queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    meta TEXT NOT NULL,
    fail_count INTEGER NOT NULL,
    status TEXT NOT NULL,
    job_type TEXT NOT NULL,
    last_msg TEXT
);

-- Index to optimize queries filtering by id
CREATE INDEX IF NOT EXISTS idx_queue_entry_id ON queue(id);
