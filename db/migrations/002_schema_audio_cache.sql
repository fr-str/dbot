
-- config
PRAGMA journal_mode=WAL;
-- NORMAL means we don't have to wait for fsync() on every write
-- we only need to wait for WAL fsync() call
PRAGMA synchronous=NORMAL;

CREATE TABLE IF NOT EXISTS audiocache (
    gid TEXT NOT NULL,
    link TEXT NOT NULL,
    title TEXT NOT NULL,
    filepath TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS gid_link on audiocache (gid,link);
