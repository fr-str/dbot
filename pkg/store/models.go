// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package store

import (
	"database/sql"

	"dbot/pkg/db/types"
)

type Channel struct {
	Gid       string
	Chid      string
	ChName    string
	Type      string
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
	DeletedAt sql.NullTime
}

type Playlist struct {
	ID         int64
	GuildID    string
	Name       string
	YoutubeUrl sql.NullString
	CreatedAt  sql.NullTime
	UpdatedAt  sql.NullTime
	DeletedAt  sql.NullTime
}

type PlaylistEntry struct {
	ID         int64
	PlaylistID int64
	YoutubeUrl string
	MinioUrl   string
	Name       string
	CreatedAt  sql.NullTime
	UpdatedAt  sql.NullTime
	DeletedAt  sql.NullTime
}

type Queue struct {
	ID        int64
	FailCount int64
	Status    string
	JobType   string
	Meta      string
	LastMsg   sql.NullString
}

type Sound struct {
	Url       string
	Gid       string
	Aliases   types.Aliases
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
	DeletedAt sql.NullTime
	OriginUrl string
}
