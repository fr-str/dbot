package player

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"dbot/pkg/config"
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

func (p *Player) Close() {
	for i := range p.list.list {
		os.Remove(p.list.list[i].Filepath)
	}
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
	err := p.fetch(a)
	if err != nil {
		p.ErrChan <- p.playerErr("failed to download", err)
		return
	}
	p.queue <- a
}

func (p *Player) musicLoop() {
	for a := range p.list.nextAudio {
		log.Debug("list.nextAudio", log.JSON(a))
		if len(a.Filepath) == 0 {
			err := p.fetch(a)
			if err != nil {
				p.ErrChan <- p.playerErr("failed to download", err)
				continue
			}
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

		if shouldUnpause {
			p.PlayPause()
		}
	}
}

func (p *Player) fetch(audio *Audio) error {
	if strings.Contains(audio.Link, p.VC.GuildID) {
		audio.Filepath = filepath.Join(config.BACKUP_DIR, audio.Link)
		return nil
	}

	meta, err := p.YTDLP.DownloadAudio(audio.Link)
	if err != nil {
		return err
	}

	audio.Filepath = meta.Filepath
	audio.Title = meta.Title
	audio.Link = meta.OriginalURL
	return nil
}

func (p *Player) play(audio *Audio) error {
	p.Playing.Store(true)
	defer p.Playing.Store(false)
	// defer os.Remove(audio.Filepath)
	log.Debug("play", log.JSON(audio))
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-i", audio.Filepath,
		"-ar", "48000",
		"-ac", "2",
		"-af", dynaudnorm,
		"-c:a", "libopus",
		"-frame_duration", "20",
		"-vbr", "off",
		"-b:a", "64k",
		"-application", "audio",
		"-packet_loss", "0",
		"-f", "opus",
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

// const dynaudnorm = `dynaudnorm=f=150:g=15:p=0.9`

const dynaudnorm = `dynaudnorm=f=100:g=5:p=0.95:m=50.0`

func (p *Player) playSound(audio *Audio) error {
	p.Playing.Store(true)
	defer p.Playing.Store(false)
	log.Debug("playSound", log.JSON(audio))
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-i", audio.Filepath,
		"-ar", "48000",
		"-ac", "2",
		"-c:a", "libopus",
		"-frame_duration", "20",
		"-af", dynaudnorm,
		"-vbr", "off",
		"-b:a", "64k",
		"-application", "audio",
		"-packet_loss", "0",
		"-f", "opus",
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
