-- name: GetChannel :one
SELECT * FROM channels
WHERE gid = ? AND type = ? LIMIT 1;

-- upsert
-- name: MapChannel :one
INSERT INTO channels (gid,chid, ch_name, type)
VALUES (:gid, :chid, :ch_name, :type)
ON CONFLICT DO UPDATE SET
    chid = excluded.chid,
    ch_name = excluded.ch_name
RETURNING *;

-- name: DeleteChannel :exec
DELETE FROM channels
WHERE chid = ?;


-- name: AddSound :one
INSERT INTO sounds (gid,url,aliases)
VALUES (:gid,:url,:aliases)
RETURNING *;

-- name: SelectSounds :many 
SELECT * FROM sounds
WHERE gid = ?;

