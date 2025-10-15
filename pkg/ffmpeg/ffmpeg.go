package ffmpeg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dbot/pkg/config"

	"github.com/fr-str/log"
)

var ErrFfmpegError = errors.New("ffmpeg error")

// file is closed when context is canceled
func ToDiscordWebm(ctx context.Context, file string) (*os.File, error) {
	tmpDir, ok := ctx.Value(config.DirKey).(string)
	if !ok || len(tmpDir) == 0 {
		return nil, errors.New("nie dałeś temp dira debilu")
	}

	mp4Path := filepath.Join(tmpDir, "discord.dupa.webm")
	info, err := Probe(file)
	if err != nil {
		return nil, err
	}

	log.Trace("ToDiscordMP4", log.String("dir", tmpDir))
	// video bitrate
	bitrate := 10 * 1_000_000 * 8
	// audio bitrate
	bitrate -= 48 * 1_000 * int(info.Format.Duration.Seconds())
	// calculate bitrate for video
	bitrate = bitrate / int(info.Format.Duration.Seconds())
	// to kbit/s
	bitrate = bitrate / 1000
	log.Trace("bitrate", log.Int("bitrate", bitrate))
	base := []string{
		"-hide_banner",
		"-i", file,
		"-c:v", "libx264",
		"-vf", "scale=-2:480",
		"-preset", "veryslow",
		"-r", "24",
		"-b:v", fmt.Sprintf("%dK", bitrate),
	}

	cmd := exec.CommandContext(ctx, "ffmpeg")
	// cmd.Dir = tmpDir
	// first pass
	cmd.Args = append(cmd.Args, base...)
	cmd.Args = append(cmd.Args,
		"-an", "-pass", "1", "-f", "mp4", "-y", "/dev/null")

	log.Info("convertToDiscordMP4 first pass", log.String("cmd", cmd.String()))
	err = runCmd(cmd)
	if err != nil {
		return nil, err
	}

	// second pass
	cmd = exec.CommandContext(ctx, "ffmpeg")
	cmd.Args = append(cmd.Args, base...)
	cmd.Args = append(cmd.Args,
		"-pass", "2",
		"-c:a", "libopus",
		"-b:a", "48k",
		"-movflags", "+faststart",
		mp4Path)

	log.Info("convertToDiscordMP4 second pass", log.String("cmd", cmd.String()))
	err = runCmd(cmd)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(mp4Path)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	log.Trace("convertToDiscordMP4",
		log.String("mp4Path", mp4Path),
		log.String("file", f.Name()), log.Int("size", stat.Size()))

	go func() {
		<-ctx.Done()
		f.Close()
	}()

	return f, nil
}

func runCmd(cmd *exec.Cmd) error {
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	cmd.Stderr = buf
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("%w: cmd.Start failed: %w", ErrFfmpegError, err)
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Println(buf.String())
		return fmt.Errorf("%w: cmd.Wait failed: %w,\n%s", ErrFfmpegError, err, buf.String())
	}

	return nil
}

// file is closed when context is canceled
func ConvertToMP4(ctx context.Context, file string) (*os.File, error) {
	tmpDir, ok := ctx.Value(config.DirKey).(string)
	if !ok || len(tmpDir) == 0 {
		return nil, errors.New("nie dałeś temp dira debilu")
	}

	buf := bytes.NewBuffer(nil)
	name := strings.ReplaceAll(file, filepath.Ext(file), "")
	mp4Path := filepath.Join(tmpDir, fmt.Sprintf("edit.%s.mp4", filepath.Base(name)))
	cmd := exec.CommandContext(ctx, "ffmpeg")
	cmd.Args = append(cmd.Args,
		"-hide_banner",
		"-init_hw_device", "qsv=hw",
		"-filter_hw_device", "hw",
		"-i", file,
		"-c:v", "h264_qsv",
		"-global_quality", "23",
		"-preset", "veryslow",
		"-movflags", "+faststart",
		mp4Path,
	)
	cmd.Stdout = buf
	cmd.Stderr = buf

	log.Info("convertToMP4", log.String("cmd", cmd.String()))
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cmd.Start failed: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("cmd.Wait failed: %w,\n%s", err, buf.String())
	}

	f, err := os.Open(mp4Path)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		f.Close()
	}()

	return f, nil
}
