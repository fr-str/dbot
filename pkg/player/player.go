package player

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"

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

func NewPlayer() *Player {
	p := &Player{
		list:    newList(),
		queue:   make(chan *Audio, 1000),
		ErrChan: make(chan Err),
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
	log.Debug("Add", log.Int("list.len", p.list.len()))
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
	meta, err := p.YTDLP.DownloadAudio(audio.Link)
	if err != nil {
		p.ErrChan <- p.playerErr("failed to download", err)
		return
	}

	audio.Filepath = meta.Filepath
	log.Trace("[dupa]", log.Any("meta.Title", meta.Title))
	audio.Title = meta.Title
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
