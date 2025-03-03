-- +goose up
-- config
-- PRAGMA journal_mode=WAL;
-- NORMAL means we don't have to wait for fsync() on every write
-- we only need to wait for WAL fsync() call
-- PRAGMA synchronous=NORMAL;

-- channels table is used for mapping discord channels
-- to bot channels, like: music, errors, admin
CREATE TABLE IF NOT EXISTS channels (
    gid TEXT NOT NULL,
    chid TEXT NOT NULL,
    ch_name TEXT NOT NULL,
    type TEXT NOT NULL,
    created_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at DATETIME DEFAULT null
);
CREATE UNIQUE INDEX IF NOT EXISTS guild_type on channels (gid,type);


CREATE TABLE IF NOT EXISTS sounds (
    url TEXT NOT NULL,
    gid TEXT NOT NULL,
    aliases Aliases NOT NULL,
    created_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at DATETIME DEFAULT null
);
CREATE UNIQUE INDEX IF NOT EXISTS url_gid ON sounds (url,gid);

