
-- name: InsertBackup :exec
INSERT INTO msg_backup (msg_id, channel_id,author_id, content, attachments, created_at)
VALUES (:msg_id, :channel_id,:author_id,  :content, :attachments, :created_at);

-- name: UpdateBackupMsg :exec
UPDATE msg_backup SET content = :content
WHERE msg_id = :msg_id;


-- name: InsertArtefact :exec
INSERT INTO artefacts (path, media_type, hash, created_at)
VALUES (:path, :media_type, :hash, :created_at);


-- name: UpsertUser :one
INSERT INTO users (discord_id, username)
VALUES (:discord_id, :username)
ON CONFLICT (discord_id) DO UPDATE SET
    username = excluded.username
RETURNING *;
