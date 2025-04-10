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
	"golang.org/x/net/context"
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
	cmd.Dir = config.TMP_PATH

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

func (YTDLP) DownloadVideo(ctx context.Context, link string) (VideoMeta, error) {
	tmpDir, ok := ctx.Value(config.DirKey).(string)
	if !ok || len(tmpDir) == 0 {
		return VideoMeta{}, errors.New("nie dałeś temp dira debilu")
	}
	cmd := exec.Command(ytdlp, append(videoDownloadCMD, link)...)
	cmd.Dir = string(tmpDir)

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
	cmd.Dir = config.TMP_PATH

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
