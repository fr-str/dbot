-- +goose up
CREATE TABLE IF NOT EXISTS playlists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    name TEXT NOT NULL,
    youtube_url TEXT,
    created_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at DATETIME DEFAULT null
);

-- Ensure that each playlist name is unique within a guild
CREATE UNIQUE INDEX IF NOT EXISTS idx_playlists_guild_id_name ON playlists(guild_id, name);

-- insert into playlists (guild_id,name) values (1,'test');
-- select * from playlists;
