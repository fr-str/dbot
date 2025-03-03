-- +goose up
CREATE TABLE IF NOT EXISTS audiocache (
    gid TEXT NOT NULL,
    link TEXT NOT NULL,
    title TEXT NOT NULL,
    filepath TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS gid_link on audiocache (gid,link);
