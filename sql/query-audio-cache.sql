-- name: SetAudio :exec
INSERT INTO audiocache (gid,link,filepath,title)
VALUES (:gid,:link,:filepath,:title);

-- name: GetAudio :one
SELECT * FROM audiocache
WHERE gid = ? and link = ?;   
