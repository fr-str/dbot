package ytdlp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

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

var ErrFailedToDownload = errors.New("yt-dlp: failed to download")

const ytdlp = "yt-dlp"

var once sync.Once

var playlistInfoCMD, videoDownloadCMD, audioDownloadCMD []string

func init() {
	go func() {
		<-config.Ctx.Done()
		audioDownloadCMD = []string{
			"--no-simulate",
			"--cookies", filepath.Join(must(os.Getwd()), "prod-data", config.COOKIE_PATH),
			"--print", "after_move:%(.{title,filepath,ext})j",
			"-x",
			"--audio-format",
			"opus",
		}
		videoDownloadCMD = []string{
			"--no-simulate",
			"--cookies", filepath.Join(must(os.Getwd()), "prod-data", config.COOKIE_PATH),
			"--print", "after_move:%(.{title,filepath,ext})j",
			"-f",
			"bestvideo+bestaudio/best",
		}

		playlistInfoCMD = []string{
			"--skip-download",
			"--flat-playlist",
			"--dump-single-json",
			"--no-colors",
			"--cookies", filepath.Join(must(os.Getwd()), "prod-data", config.COOKIE_PATH),
		}
	}()
}

type VideoMeta struct {
	Title    string
	Ext      string
	Filepath string
}

func (YTDLP) DownloadAudio(link string) (VideoMeta, error) {
	cmd := exec.Command(ytdlp, append(audioDownloadCMD, link)...)
	cmd.Dir = config.YTDLP_DOWNLOAD_DIR

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var meta VideoMeta
	log.Trace("DownloadAudio", log.String("cmd", cmd.String()), log.String("link", link))
	err := cmd.Run()
	if err != nil {
		b, _ := io.ReadAll(stderr)
		return meta, errors.Join(ErrFailedToDownload, errors.New(string(b)))
	}

	err = json.NewDecoder(stdout).Decode(&meta)
	if err != nil {
		return meta, err
	}

	return meta, nil
}

func (YTDLP) DownloadVideo(link string) (VideoMeta, error) {
	cmd := exec.Command(ytdlp, append(videoDownloadCMD, link)...)
	cmd.Dir = config.YTDLP_DOWNLOAD_DIR

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var meta VideoMeta
	log.Trace("DownloadVideo", log.String("cmd", cmd.String()), log.String("link", link))
	err := cmd.Run()
	if err != nil {
		b, _ := io.ReadAll(stderr)
		return meta, errors.Join(ErrFailedToDownload, errors.New(string(b)))
	}

	err = json.NewDecoder(stdout).Decode(&meta)
	if err != nil {
		return meta, err
	}

	return meta, nil
}

type PlaylistMeta struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	WebpageURL string `json:"webpage_url"`
	Entries    []struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Title    string `json:"title"`
		Duration *int   `json:"duration"`
	} `json:"entries"`
}

func (YTDLP) PlaylistInfo(link string) (PlaylistMeta, error) {
	cmd := exec.Command(ytdlp, append(playlistInfoCMD, link)...)
	cmd.Dir = config.YTDLP_DOWNLOAD_DIR

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var meta PlaylistMeta
	log.Trace("PlaylistInfo", log.String("cmd", cmd.String()), log.String("link", link))
	err := cmd.Run()
	if err != nil {
		b, _ := io.ReadAll(stderr)
		return meta, errors.Join(ErrFailedToDownload, errors.New(string(b)))
	}

	err = json.NewDecoder(stdout).Decode(&meta)
	if err != nil {
		return meta, err
	}

	return meta, nil
}
