package player

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"dbot/pkg/cache"
	"dbot/pkg/dbg"
	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
	"github.com/pion/opus/pkg/oggreader"
)

func (p *Player) playerErr(msg string, vars ...any) Err {
	return Err{
		Err: fmt.Errorf("player: "+msg+": %w", vars...),
		GID: p.VC.GuildID,
	}
}

func (t *Player) PlayPause() {
	if !t.paused {
		t.Lock()
		t.paused = true
		t.Playing.Store(false)
		log.Trace("t.paused", log.Bool("paused", t.paused))
		return
	}

	t.Playing.Store(true)
	t.paused = false
	t.Unlock()
	log.Trace("t.paused", log.Bool("paused", t.paused))
}

type Player struct {
	ytdlp.YTDLP
	cache *cache.Queries

	list    list
	VC      *discordgo.VoiceConnection
	ErrChan chan Err

	// play pause
	sync.Mutex
	paused  bool
	Playing atomic.Bool

	// soundboard
	queue chan *Audio
}

type Err struct {
	GID string
	Err error
}

func NewPlayer(cache *cache.Queries) *Player {
	p := &Player{
		list:    newList(),
		queue:   make(chan *Audio, 1000),
		ErrChan: make(chan Err),
		cache:   cache,
	}
	go p.musicLoop()
	go p.soundLoop()

	return p
}

func (p *Player) Current() *Audio {
	return p.list.current()
}

func (p *Player) Next() *Audio {
	return p.list.peek()
}

func (p *Player) Add(link string) {
	p.list.add(link)
	if p.paused {
		p.PlayPause()
	}
}

func (p *Player) PlaySound(link string) {
	a := &Audio{Link: link}
	p.fetch(a)
	p.queue <- a
}

func (p *Player) musicLoop() {
	for a := range p.list.nextAudio {
		log.Debug("list.nextAudio", log.JSON(a))
		if len(a.Filepath) == 0 {
			p.fetch(a)
		}

		err := p.play(a)
		if err != nil {
			p.ErrChan <- p.playerErr("failed to play", err)
			continue
		}

		os.Remove(a.Filepath)

		log.Trace("musicLoop", log.Bool("p.list.more()", p.list.more()))
		if !p.list.more() {
			p.PlayPause()
		}

		// Locking allows us to wait if p.list.more() returns false
		// after user add aditional entry p.PlayPause will be trigered unlocking the mutex
		p.Lock()
		p.list.next()
		p.Unlock()
	}
}

func (p *Player) soundLoop() {
	for a := range p.queue {
		var shouldUnpause bool
		if !p.paused {
			shouldUnpause = true
			p.PlayPause()
		}
		err := p.playSound(a)
		if err != nil {
			p.ErrChan <- p.playerErr("failed to play", err)
			continue
		}

		log.Trace("soundLoop", log.Any("shouldUnpause", shouldUnpause))
		if shouldUnpause {
			p.PlayPause()
		}
	}
}

func (p *Player) fetch(audio *Audio) {
	czary := czaryMaryŻebyDziałało(audio.Link)
	dbg.Assert(p.VC != nil, "nil VC")
	au, err := p.cache.GetAudio(context.Background(), cache.GetAudioParams{
		Gid:  p.VC.GuildID,
		Link: czary,
	})
	if err == nil {
		log.Trace("audio cache HIT", log.JSON(au))
		audio.Filepath = au.Filepath
		audio.Title = au.Title
		return
	}
	log.Trace("audio cache MISS", log.String("link", audio.Link), log.String("gid", p.VC.GuildID))

	meta, err := p.YTDLP.DownloadAudio(audio.Link)
	if err != nil {
		p.ErrChan <- p.playerErr("failed to download", err)
		return
	}

	audio.Filepath = meta.Filepath
	audio.Title = meta.Title
	// TODO: temprary, will probably store everything in minio
	if !strings.Contains(audio.Link, "youtu") {
		err = p.cache.SetAudio(context.Background(), cache.SetAudioParams{
			Gid:      p.VC.GuildID,
			Link:     czary,
			Filepath: audio.Filepath,
			Title:    audio.Title,
		})
		if err != nil {
			log.Warn("failed to set in cache", log.Err(err))
		}
	}
}

func czaryMaryŻebyDziałało(link string) string {
	if strings.Contains(link, "youtu") {
		return link
	}

	url, _, found := strings.Cut(link, "?")
	if !found {
		return link
	}

	return url
}

func (p *Player) play(audio *Audio) error {
	p.Playing.Store(true)
	defer p.Playing.Store(false)
	log.Debug("play", log.JSON(audio))
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-i", audio.Filepath,
		"-ar", "48000", // Sample rate for Opus
		"-ac", "2", // Stereo
		"-c:a", "libopus", // Opus codec
		"-frame_duration", "20", // 20ms frames
		"-vbr", "off", // Disable variable bitrate for consistent frame sizes
		"-b:a", "64k", // Bitrate
		"-application", "audio",
		"-packet_loss", "0", // Disable packet loss prevention
		"-f", "opus", // Force opus format
		"pipe:1",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create Stdout pipe: %w", err)
	}

	log.Info("play", log.String("cmd", cmd.String()))
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd.Start failed: %w", err)
	}

	reader, _, err := oggreader.NewWith(stdout)
	if err != nil {
		return fmt.Errorf("failed to create oggreader from stdout: %w", err)
	}

	p.VC.Speaking(true)
	defer p.VC.Speaking(false)
	for {
		page, _, err := reader.ParseNextPage()
		if err != nil {
			if err != io.EOF {
				log.Error("failed to parse page", log.Err(err))
			}
			break
		}

		for _, frame := range page {
			p.Lock()
			p.VC.OpusSend <- frame
			p.Unlock()
		}
	}

	// Wait for FFmpeg to finish
	return cmd.Wait()
}

func (p *Player) playSound(audio *Audio) error {
	p.Playing.Store(true)
	defer p.Playing.Store(false)
	log.Debug("playSound", log.JSON(audio))
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-i", audio.Filepath,
		"-ar", "48000", // Sample rate for Opus
		"-ac", "2", // Stereo
		"-c:a", "libopus", // Opus codec
		"-frame_duration", "20", // 20ms frames
		"-vbr", "off", // Disable variable bitrate for consistent frame sizes
		"-b:a", "64k", // Bitrate
		"-application", "audio",
		"-packet_loss", "0", // Disable packet loss prevention
		"-f", "opus", // Force opus format
		"pipe:1",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create Stdout pipe: %w", err)
	}

	log.Info("playSound", log.String("cmd", cmd.String()))
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd.Start failed: %w", err)
	}

	reader, _, err := oggreader.NewWith(stdout)
	if err != nil {
		return fmt.Errorf("failed to create oggreader from stdout: %w", err)
	}

	p.VC.Speaking(true)
	defer p.VC.Speaking(false)
	for {
		page, _, err := reader.ParseNextPage()
		if err != nil {
			if err != io.EOF {
				log.Error("failed to parse page", log.Err(err))
			}
			break
		}

		for _, frame := range page {
			p.VC.OpusSend <- frame
		}
	}

	// Wait for FFmpeg to finish
	return cmd.Wait()
}
