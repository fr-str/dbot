
-- name: InsertBackup :exec
INSERT INTO msg_backup (msg_id, channel_id,author_id, content, attachments, created_at)
VALUES (:msg_id, :channel_id,:author_id,  :content, :attachments, :created_at);

-- name: UpdateBackupMsg :exec
UPDATE msg_backup SET content = :content
WHERE msg_id = :msg_id;


-- name: InsertArtefact :exec
INSERT INTO artefacts (origin_url,path, media_type, hash, created_at,gid,chid,msgid)
VALUES (:origin_url,:path, :media_type, :hash, :created_at, :gid, :chid, :msgid);

-- name: GetArtefacts :many
SELECT * FROM artefacts 
WHERE 
    media_type in ('image/jpg','image/png') 
AND 
    gid = ?
ORDER BY created_at DESC LIMIT 100 OFFSET :offset;

-- name: DeleteArtefact :exec
DELETE FROM artefacts WHERE gid = :gid AND chid = :chid and msgid = :msgid;

-- name: GetArtefact :one
SELECT * FROM artefacts WHERE origin_url = :origin_url;


-- name: UpsertUser :one
INSERT INTO users (discord_id, username)
VALUES (:discord_id, :username)
ON CONFLICT (discord_id) DO UPDATE SET
    username = excluded.username
RETURNING *;
