-- -- name: InsetBackup :exec
-- INSERT INTO backup (msg_id, author_id, content, attachments, created_at)
-- VALUES (:msg_id, :author_id, :content, :attachments, :created_at)
--     RETURNING *;
--
-- -- name: UpdateBackupMsg :exec
-- UPDATE backup
-- SET content = :content, attachments = :attachments
-- WHERE msg_id = :msg_id;

