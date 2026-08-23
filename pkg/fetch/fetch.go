package fetch

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"

	// Decoders
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

var downloader ytdlp.YTDLP

type extractor func(ctx context.Context, url string) ([]*discordgo.File, error)

var extractors = map[string]extractor{
	"jbzd.com.pl":   jbzd,
	"m.jbzd.com.pl": jbzd,
}

func Fetch(ctx context.Context, link string) ([]*discordgo.File, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	u, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("invalid extraction url: %w", err)
	}

	extract, ok := extractors[u.Host]
	if ok {
		return extract(ctx, link)
	}

	file, err := downloadToRam(ctx, link)
	if err != nil {
		return nil, err
	}
	return []*discordgo.File{file}, nil
}

func get(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func getImage(ctx context.Context, url string) (image.Image, error) {
	body, err := get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	img, _, err := image.Decode(body)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func readerToRam(r io.ReadCloser) io.Reader {
	b := &bytes.Buffer{}
	io.Copy(b, r)
	r.Close()
	return b
}

func downloadToRam(ctx context.Context, url string) (*discordgo.File, error) {
	info, err := downloader.DownloadVideoSmall(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("downloadToRam: %w", err)
	}

	f, err := os.Open(info.Filepath)
	if err != nil {
		return nil, fmt.Errorf("downloadToRam: open %w", err)
	}

	return &discordgo.File{Name: "dupa." + info.Ext, Reader: readerToRam(f)}, nil
}
