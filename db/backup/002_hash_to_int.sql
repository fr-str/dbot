-- +goose up
ALTER TABLE artefacts RENAME TO artefacts_old;

CREATE TABLE artefacts (
    origin_url TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    media_type TEXT NOT NULL,
    hash INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    gid TEXT NOT NULL,
    chid TEXT NOT NULL,
    msgid TEXT NOT NULL
);

INSERT INTO artefacts (origin_url, path, media_type, hash, created_at)
SELECT 
    origin_url, 
    path, 
    media_type, 
    0,
    created_at
FROM artefacts_old;

DROP TABLE artefacts_old;