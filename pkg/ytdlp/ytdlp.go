package ytdlp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"dbot/pkg/config"

	"github.com/fr-str/log"
)

type YTDLP struct{}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

type VideoMeta struct {
	Title    string
	Ext      string
	Filepath string
}

const ytdlp = "yt-dlp"

var audioDownloadCMD = []string{
	"--no-simulate",
	"--cookies", filepath.Join(must(os.Getwd()), config.COOKIE_PATH),
	"--print", "after_move:%(.{title,filepath,ext})j",
	"-x",
	"--audio-format",
	"opus",
}

func (YTDLP) DownloadAudio(link string) (VideoMeta, error) {
	cmd := exec.Command(ytdlp, append(audioDownloadCMD, link)...)
	cmd.Dir = config.YTDLP_DOWNLOAD_DIR

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var meta VideoMeta
	log.Info("DownloadAudio", log.String("cmd", cmd.String()), log.String("link", link))
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

var videoDownloadCMD = []string{
	"--no-simulate",
	"--cookies", filepath.Join(must(os.Getwd()), config.COOKIE_PATH),
	"--print", "after_move:%(.{title,filepath,ext})j",
	"-f",
	"bestvideo+bestaudio/best",
}

func (YTDLP) DownloadVideo(link string) (VideoMeta, error) {
	cmd := exec.Command(ytdlp, append(videoDownloadCMD, link)...)
	cmd.Dir = config.YTDLP_DOWNLOAD_DIR

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var meta VideoMeta
	log.Info("DownloadVideo", log.String("cmd", cmd.String()), log.String("link", link))
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
