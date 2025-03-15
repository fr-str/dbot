-- +goose up
ALTER TABLE sounds ADD origin_url TEXT NOT NULL DEFAULT '-';
