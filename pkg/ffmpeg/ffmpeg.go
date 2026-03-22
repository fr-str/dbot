package ffmpeg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dbot/pkg/config"

	"github.com/fr-str/log"
)

var ErrFfmpegError = errors.New("ffmpeg error")

// file is closed when context is canceled
func ToDiscordMP4(ctx context.Context, file string, mute bool, clip Clip) (*os.File, error) {
	tmpDir, ok := ctx.Value(config.DirKey).(string)
	if !ok || len(tmpDir) == 0 {
		return nil, errors.New("nie dałeś temp dira debilu")
	}

	mp4Path := filepath.Join(tmpDir, "discord.dupa.mp4")
	info, err := Probe(file)
	if err != nil {
		return nil, err
	}

	log.Trace("ToDiscordMP4", log.String("dir", tmpDir))
	duration := info.Format.Duration.Seconds()
	if clip.End > 0 {
		duration = clip.End.Seconds()
	}
	if clip.Start > 0 {
		duration -= clip.Start.Seconds()
	}
	if duration <= 0 {
		return nil, errors.New("invalid clip duration")
	}
	bitrate := 10 * 1_000_000 * 8
	bitrate -= 48 * 1_000 * int(duration)
	bitrate = bitrate / int(duration)
	bitrate = bitrate / 1000
	log.Trace("bitrate", log.Int("bitrate", bitrate))
	base := []string{
		"-hide_banner",
	}
	if clip.Start > 0 {
		base = append(base, "-ss", clip.Start.String())
	}
	base = append(base, "-i", file)
	if clip.End > 0 {
		base = append(base, "-t", fmt.Sprintf("%.2f", clip.End.Seconds()-clip.Start.Seconds()))
	}
	base = append(base,
		"-c:v", "libx264",
		"-vf", "scale=-2:480",
		"-preset", "veryslow",
		"-r", "24",
		"-b:v", fmt.Sprintf("%dK", bitrate),
	)

	cmd := exec.CommandContext(ctx, "ffmpeg")
	cmd.Args = append(cmd.Args, base...)
	cmd.Args = append(cmd.Args,
		"-an", "-pass", "1", "-f", "mp4", "-y", "/dev/null")

	log.Info("convertToDiscordMP4 first pass", log.String("cmd", cmd.String()))
	err = runCmd(cmd)
	if err != nil {
		return nil, err
	}

	cmd = exec.CommandContext(ctx, "ffmpeg")
	cmd.Args = append(cmd.Args, base...)
	if mute {
		cmd.Args = append(cmd.Args,
			"-pass", "2",
			"-an",
			"-movflags", "+faststart",
			mp4Path)
	} else {
		cmd.Args = append(cmd.Args,
			"-pass", "2",
			"-c:a", "libopus",
			"-b:a", "48k",
			"-movflags", "+faststart",
			mp4Path)
	}

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

type Clip struct {
	Start time.Duration
	End   time.Duration
}

type GifSettings struct {
	Height int
	FPS    int
	Clip   Clip
}

func parseTime(s string) (time.Duration, error) {
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		var total time.Duration
		for i, p := range parts {
			v, err := strconv.Atoi(p)
			if err != nil {
				return 0, err
			}
			switch len(parts) - i {
			case 3:
				total += time.Duration(v) * time.Hour
			case 2:
				total += time.Duration(v) * time.Minute
			case 1:
				total += time.Duration(v) * time.Second
			}
		}
		return total, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		d, err = time.ParseDuration(s + "s")
		if err != nil {
			return 0, err
		}
	}
	return d, nil
}

// file is closed when context is canceled
func ToDiscordGIF(ctx context.Context, file string, settings GifSettings) (*os.File, error) {
	tmpDir, ok := ctx.Value(config.DirKey).(string)
	if !ok || len(tmpDir) == 0 {
		return nil, errors.New("nie dałeś temp dira debilu")
	}

	gifPath := filepath.Join(tmpDir, "discord.dupa.gif")
	info, err := Probe(file)
	if err != nil {
		return nil, err
	}

	log.Trace("ToDiscordGIF", log.String("dir", tmpDir))

	duration := info.Format.Duration.Seconds()
	if settings.Clip.End > 0 {
		duration = settings.Clip.End.Seconds()
	}
	if settings.Clip.Start > 0 {
		duration -= settings.Clip.Start.Seconds()
	}

	filter := fmt.Sprintf(
		"scale=-2:%d,fps=%d,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle",
		settings.Height, settings.FPS,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg")
	cmd.Args = append(cmd.Args,
		"-hide_banner")
	if settings.Clip.Start > 0 {
		cmd.Args = append(cmd.Args, "-ss", settings.Clip.Start.String())
	}
	cmd.Args = append(cmd.Args,
		"-i", file,
		"-t", fmt.Sprintf("%.2f", duration),
		"-vf", filter,
		"-y",
		gifPath,
	)

	log.Info("convertToDiscordGIF", log.String("cmd", cmd.String()))
	err = runCmd(cmd)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(gifPath)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	log.Trace("convertToDiscordGIF",
		log.String("gifPath", gifPath),
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
func ConvertToMP4(ctx context.Context, file string, clip Clip) (*os.File, error) {
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
		"-filter_hw_device", "hw")
	if clip.Start > 0 {
		cmd.Args = append(cmd.Args, "-ss", clip.Start.String())
	}
	cmd.Args = append(cmd.Args, "-i", file)
	if clip.End > 0 {
		cmd.Args = append(cmd.Args, "-t", fmt.Sprintf("%.2f", clip.End.Seconds()-clip.Start.Seconds()))
	}
	cmd.Args = append(cmd.Args,
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
