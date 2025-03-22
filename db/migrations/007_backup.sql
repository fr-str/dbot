-- +goose up
CREATE TABLE IF NOT EXISTS backup (
    msg_id INTEGER NOT NULL PRIMARY KEY,
    author_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    attachments TEXT NOT NULL,
    created_at DATETIME NOT NULL
);


