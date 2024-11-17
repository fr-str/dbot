package ytdlp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"

	"dbot/pkg/config"

	"github.com/fr-str/log"
)

type YTDLP struct{}

const ytdlp = "yt-dlp"

var audioDownloadCMD = []string{
	"--no-simulate",
	"--print", "after_move:%(.{title,filepath,ext})j",
	"-x",
	"--audio-format",
	"opus",
}

type VideoMeta struct {
	Title    string
	Ext      string
	Filepath string
}

func (YTDLP) DownloadAudio(link string) (VideoMeta, error) {
	log.Debug("DownloadAudio", log.String("link", link))
	cmd := exec.Command(ytdlp, append(audioDownloadCMD, link)...)
	cmd.Dir = config.YTDLP_DOWNLOAD_DIR

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var meta VideoMeta
	log.Info("DownloadAudio", log.String("cmd", cmd.String()))
	err := cmd.Run()
	if err != nil {
		b, _ := io.ReadAll(stderr)
		return meta, errors.New(string(b))
	}

	err = json.NewDecoder(stdout).Decode(&meta)
	if err != nil {
		return meta, err
	}

	return meta, nil
}
