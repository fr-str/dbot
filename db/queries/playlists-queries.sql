-- Insert a new playlist
-- name: CreatePlaylist :one
INSERT INTO playlists (guild_id, name, youtube_url)
VALUES (?, ?, ?) RETURNING *;

-- Retrieve a playlist by guild_id and name
-- name: GetPlaylist :one
SELECT id, guild_id, name, youtube_url, created_at, updated_at
FROM playlists
WHERE guild_id = ? AND name = ?
LIMIT 1;

-- Insert a new playlist entry
-- name: AddPlaylistEntry :one
INSERT INTO playlist_entries (playlist_id, youtube_url, minio_url, name)
VALUES (?, ?, ?, ?) RETURNING *;

-- Retrieve all entries for a given playlist
-- name: ListPlaylistEntries :many
SELECT id, playlist_id, youtube_url, minio_url, name, created_at, updated_at
FROM playlist_entries
WHERE playlist_id = ?
ORDER BY created_at;
