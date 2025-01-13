package sql

import _ "embed"

//go:embed db-schema.sql
var DBSchema string

//go:embed schema-audio-cache.sql
var AudioCacheSchema string
