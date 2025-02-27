package ytdlp

import (
	"context"
	"os/exec"
	"time"

	"github.com/fr-str/log"
)

func StartUpdater(ctx context.Context) {
	tic := time.NewTicker(time.Hour)
	version, err := exec.Command("yt-dlp", "--version").CombinedOutput()
	if err != nil {
		log.Error("StartUpdater.yt-dlp -U",
			log.String("timestamp", time.Now().String()),
			log.Err(err),
			log.String("out", string(version)))
	}
	log.Info("yt-dlp", log.String("version", string(version)))
	go func() {
		for {
			out, err := exec.Command("yt-dlp", "-U").CombinedOutput()
			if err != nil {
				log.Error("StartUpdater.yt-dlp -U",
					log.String("timestamp", time.Now().String()),
					log.Err(err),
					log.String("out", string(out)))
			}

			versionNew, err := exec.Command("yt-dlp", "--version").CombinedOutput()
			if err != nil {
				log.Error("StartUpdater.yt-dlp -U",
					log.String("timestamp", time.Now().String()),
					log.Err(err),
					log.String("out", string(version)))
			}
			if string(version) != string(versionNew) {
				log.Info("updated yt-dlp", log.String("old", string(version)), log.String("new", string(versionNew)))
			}

			select {
			case <-tic.C:
			case <-ctx.Done():
				return
			}
		}
	}()
}
