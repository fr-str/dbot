package player

import (
	"fmt"
	"io"
	"os/exec"
	"sync"

	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
	"github.com/pion/opus/pkg/oggreader"
)

func playerErr(msg string, vars ...any) error {
	return fmt.Errorf("player: "+msg+": %w", vars...)
}

type toggle struct {
	sync.Mutex
	paused bool
}

func (t *toggle) PlayPause() {
	if t.paused {
		t.paused = false
		t.Unlock()
		log.Debug("t.paused", log.Bool("paused", t.paused))
		return
	}
	t.Lock()
	t.paused = true
	log.Debug("t.paused", log.Bool("paused", t.paused))
}

type Player struct {
	ytdlp.YTDLP

	list    list
	pause   toggle
	VC      *discordgo.VoiceConnection
	ErrChan chan error
}

func NewPlayer() *Player {
	p := &Player{
		list:    newList(),
		pause:   toggle{},
		ErrChan: make(chan error),
	}
	go p.loop()

	return p
}

func (p *Player) Add(link string) {
	p.list.add(link)
	log.Debug("Add", log.Int("list.len", p.list.len()))
	if p.list.len() == 1 {
		p.list.next()
	}
}

func (p *Player) loop() {
	for a := range p.list.nextAudio {
		p.fetch(p.list.peek())
		err := p.play(a)
		if err != nil {
			p.ErrChan <- playerErr("failed to play", err)
		}
		p.list.next()
	}
}

func (p *Player) fetch(audio *Audio) {
	meta, err := p.YTDLP.DownloadAudio(audio.Link)
	if err != nil {
		p.ErrChan <- playerErr("failed to download", err)
		return
	}

	audio.Filepath = meta.Filepath
}

func (p *Player) play(audio *Audio) error {
	log.Debug("playFromFile", log.JSON(audio))
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
		return playerErr("failed to create Stdout pipe", err)
	}

	log.Info("playFromFile", log.String("cmd", cmd.String()))
	err = cmd.Start()
	if err != nil {
		return playerErr("cmd.Start failed", err)
	}

	reader, _, err := oggreader.NewWith(stdout)
	if err != nil {
		return playerErr("failed to create oggreader from stdout", err)
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
			p.pause.Lock()
			p.VC.OpusSend <- frame
			p.pause.Unlock()
		}
	}

	// Wait for FFmpeg to finish
	return cmd.Wait()
}
