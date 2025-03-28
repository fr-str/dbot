package config

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fr-str/env"
)

const (
	Prod = "prod"
)

var (
	// when config is done loading this context is canceled
	// and all <-config.Ctx.Done() will be unblocked
	Ctx, cancel = context.WithCancel(context.Background())

	ENV    string
	DB_DIR string

	TOKEN    string
	GUILD_ID string

	YTDLP_DOWNLOAD_DIR string
	COOKIE_PATH        string

	FFMPEG_TRANSCODE_PATH string
	FFMPEG_HW_ACCEL       bool

	BACKUP_DIR string

	// minio
	MINIO_HOST              string
	MINIO_ACCESS_KEY_ID     string
	MINIO_SECRET_ACCESS_KEY string
	MINIO_DBOT_BUCKET_NAME  string
)

func Load() {
	defer cancel()
	ENV = env.Get("ENV", "dev")
	DB_DIR = env.Get("DB_DIR", "data")

	TOKEN = env.Get("TOKEN", "")
	GUILD_ID = env.Get("GUILD_ID", "")

	YTDLP_DOWNLOAD_DIR = env.Get("YTDLP_DOWNLOAD_DIR", filepath.Join(os.TempDir(), "dbot"))
	COOKIE_PATH = env.Get("COOKIE_PATH", "")

	FFMPEG_TRANSCODE_PATH = env.Get("FFMPEG_TRANSCODE_PATH", filepath.Join(os.TempDir(), "dbot", "ffmpeg"))
	FFMPEG_HW_ACCEL = env.Get("FFMPEG_HW_ACCEL", false)

	BACKUP_DIR = env.Get("BACKUP_DIR", filepath.Join(os.TempDir(), "dbot", "backup"))

	dirs := []string{
		YTDLP_DOWNLOAD_DIR,
		FFMPEG_TRANSCODE_PATH,
		BACKUP_DIR,
		DB_DIR,
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
