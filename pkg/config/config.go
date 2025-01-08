package config

import (
	"os"
	"path/filepath"

	"github.com/fr-str/env"
)

var (
	TOKEN    = env.Get("TOKEN", "")
	GUILD_ID = env.Get("GUILD_ID", "")

	YTDLP_DOWNLOAD_DIR = env.Get("YTDLP_DOWNLOAD_DIR", filepath.Join(os.TempDir(), "dbot"))
	COOKIE_PATH        = env.Get("COOKIE_PATH", "")

	FFMPEG_TRANSCODE_PATH = env.Get("FFMPEG_TRANSCODE_PATH", filepath.Join(os.TempDir(), "dbot", "ffmpeg"))

	// minio
	MINIO_HOST              = env.Get[string]("MINIO_HOST")
	MINIO_ACCESS_KEY_ID     = env.Get[string]("MINIO_ACCESS_KEY_ID")
	MINIO_SECRET_ACCESS_KEY = env.Get[string]("MINIO_SECRET_ACCESS_KEY")
	MINIO_DBOT_BUCKET_NAME  = env.Get("MINIO_DBOT_BUCKET_NAME", "dbot")
)

func init() {
	dirs := []string{
		YTDLP_DOWNLOAD_DIR,
		FFMPEG_TRANSCODE_PATH,
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			continue
		}

		err := os.MkdirAll(dir, 0o777)
		if err != nil {
			panic(err)
		}
	}
}
