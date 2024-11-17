package config

import (
	"os"
	"path/filepath"

	"github.com/fr-str/env"
)

var (
	TOKEN    = env.Get[string]("TOKEN")
	GUILD_ID = env.Get("GUILD_ID", "")

	YTDLP_DOWNLOAD_DIR = env.Get("YTDLP_DOWNLOAD_DIR", filepath.Join(os.TempDir(), "dbot"))
)

func init() {
	dirs := []string{
		YTDLP_DOWNLOAD_DIR,
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
