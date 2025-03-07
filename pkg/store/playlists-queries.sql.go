// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: playlists-queries.sql

package store

import (
	"context"
	"database/sql"
)

const addPlaylistEntry = `-- name: AddPlaylistEntry :one
INSERT INTO playlist_entries (playlist_id, youtube_url, minio_url, name)
VALUES (?, ?, ?, ?) RETURNING id, playlist_id, youtube_url, minio_url, name, created_at, updated_at, deleted_at
`

type AddPlaylistEntryParams struct {
	PlaylistID int64
	YoutubeUrl string
	MinioUrl   string
	Name       string
}

// Insert a new playlist entry
func (q *Queries) AddPlaylistEntry(ctx context.Context, arg AddPlaylistEntryParams) (PlaylistEntry, error) {
	row := q.db.QueryRowContext(ctx, addPlaylistEntry,
		arg.PlaylistID,
		arg.YoutubeUrl,
		arg.MinioUrl,
		arg.Name,
	)
	var i PlaylistEntry
	err := row.Scan(
		&i.ID,
		&i.PlaylistID,
		&i.YoutubeUrl,
		&i.MinioUrl,
		&i.Name,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const createPlaylist = `-- name: CreatePlaylist :one
INSERT INTO playlists (guild_id, name, youtube_url)
VALUES (?, ?, ?) RETURNING id, guild_id, name, youtube_url, created_at, updated_at, deleted_at
`

type CreatePlaylistParams struct {
	GuildID    string
	Name       string
	YoutubeUrl sql.NullString
}

// Insert a new playlist
func (q *Queries) CreatePlaylist(ctx context.Context, arg CreatePlaylistParams) (Playlist, error) {
	row := q.db.QueryRowContext(ctx, createPlaylist, arg.GuildID, arg.Name, arg.YoutubeUrl)
	var i Playlist
	err := row.Scan(
		&i.ID,
		&i.GuildID,
		&i.Name,
		&i.YoutubeUrl,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const getPlaylist = `-- name: GetPlaylist :one
SELECT id, guild_id, name, youtube_url, created_at, updated_at
FROM playlists
WHERE guild_id = ? AND name = ? AND deleted_at IS NULL
LIMIT 1
`

type GetPlaylistParams struct {
	GuildID string
	Name    string
}

type GetPlaylistRow struct {
	ID         int64
	GuildID    string
	Name       string
	YoutubeUrl sql.NullString
	CreatedAt  sql.NullTime
	UpdatedAt  sql.NullTime
}

// Retrieve a playlist by guild_id and name
func (q *Queries) GetPlaylist(ctx context.Context, arg GetPlaylistParams) (GetPlaylistRow, error) {
	row := q.db.QueryRowContext(ctx, getPlaylist, arg.GuildID, arg.Name)
	var i GetPlaylistRow
	err := row.Scan(
		&i.ID,
		&i.GuildID,
		&i.Name,
		&i.YoutubeUrl,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listPlaylistEntries = `-- name: ListPlaylistEntries :many
SELECT id, playlist_id, youtube_url, minio_url, name, created_at, updated_at
FROM playlist_entries
WHERE playlist_id = ? AND deleted_at IS NULL
ORDER BY created_at
`

type ListPlaylistEntriesRow struct {
	ID         int64
	PlaylistID int64
	YoutubeUrl string
	MinioUrl   string
	Name       string
	CreatedAt  sql.NullTime
	UpdatedAt  sql.NullTime
}

// Retrieve all entries for a given playlist
func (q *Queries) ListPlaylistEntries(ctx context.Context, playlistID int64) ([]ListPlaylistEntriesRow, error) {
	rows, err := q.db.QueryContext(ctx, listPlaylistEntries, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPlaylistEntriesRow
	for rows.Next() {
		var i ListPlaylistEntriesRow
		if err := rows.Scan(
			&i.ID,
			&i.PlaylistID,
			&i.YoutubeUrl,
			&i.MinioUrl,
			&i.Name,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const playlistNames = `-- name: PlaylistNames :many
SELECT name
FROM playlists
WHERE guild_id = ? AND deleted_at IS NULL
LIMIT 10
`

// Retrieve a playlist names by guild_id
func (q *Queries) PlaylistNames(ctx context.Context, guildID string) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, playlistNames, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
