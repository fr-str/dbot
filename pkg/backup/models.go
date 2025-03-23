// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package backup

import (
	"time"
)

type Artefact struct {
	ID        int64
	Path      string
	MediaType string
	Hash      string
	CreatedAt time.Time
}

type MsgBackup struct {
	MsgID       int64
	ChannelID   int64
	AuthorID    int64
	Content     string
	Attachments string
	CreatedAt   time.Time
}

type User struct {
	DiscordID int64
	Username  string
}
