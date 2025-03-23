-- +goose up
-- we are changing sound.url from dbot,<GUID>/sounds/name.mp4 to <GUID>/sounds/name.mp4

UPDATE sounds
SET url = REPLACE(url, 'dbot,', '')
WHERE url LIKE 'dbot,%';

-- playlist_entries.minio_url is now playlist_entries.filepath
ALTER TABLE playlist_entries
RENAME COLUMN minio_url TO filepath;

UPDATE playlist_entries
SET filepath = REPLACE(filepath, 'dbot,', '')
WHERE filepath LIKE 'dbot,%';

