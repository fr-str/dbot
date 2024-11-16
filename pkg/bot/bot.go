package dbot

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"strconv"

	"dbot/pkg/config"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
	"layeh.com/gopus"
)

type DBot struct {
	*discordgo.Session
}

func Start(d *discordgo.Session) {
	b := DBot{d}

	d.AddHandler(b.Ready)
	err := d.Open()
	if err != nil {
		panic(err)
	}

	cmdHandlers := b.CommandHandlers()
	d.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := cmdHandlers[i.ApplicationCommandData().Name]; ok {
			err := h(s, i)
			if err != nil {
				log.Error(err.Error())
			}
		}
	})

	for _, v := range cmds {
		_, err := d.ApplicationCommandCreate(d.State.User.ID, config.GUILD_ID, v)
		if err != nil {
			panic(err)
		}
	}
}

func (d *DBot) Ready(s *discordgo.Session, e *discordgo.Ready) {
	log.Info("ready")
}

func (d *DBot) getUserVC(s *discordgo.Session, gID string, uID string) (*discordgo.VoiceState, error) {
	g, err := s.State.Guild(gID)
	if err != nil {
		return nil, err
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID != uID {
			continue
		}
		return vs, nil
	}

	return nil, nil
}

func (d *DBot) handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	channel, err := d.getUserVC(s, i.GuildID, i.Member.User.ID)
	if err != nil {
		return err
	}

	vc, err := s.ChannelVoiceJoin(i.GuildID, channel.ChannelID, false, false)
	if err != nil {
		return err
	}

	vc.Speaking(true)
	defer vc.Speaking(false)
	err = playFromFile("./dupa.opus", vc.OpusSend)
	if err != nil {
		return err
	}

	return nil
}

const (
	channels  = 2
	sampling  = 48000
	frameSize = 960
	maxBytes  = (frameSize * 2) * 2
)

func playFromFile(fileName string, vcChan chan<- []byte) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	cmd := exec.Command("ffmpeg", "-i", fileName, "-f", "s16le", "-ar", strconv.Itoa(sampling), "-ac", strconv.Itoa(channels), "pipe:1")
	ffOut, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}
	defer cmd.Process.Kill()

	pcm := make(chan []int16, 2)
	defer close(pcm)

	go func() {
		err := sendPCM(vcChan, pcm)
		if err != nil {
			log.Error(err.Error())
		}
	}()

	ffmpegBuf := bufio.NewReaderSize(ffOut, 1<<14)
	for {
		audioBuf := make([]int16, frameSize*channels)
		err = binary.Read(ffmpegBuf, binary.LittleEndian, audioBuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		}
		if err != nil {
			return err
		}

		select {
		case pcm <- audioBuf:
		}
	}
}

func sendPCM(vcChan chan<- []byte, pcm <-chan []int16) error {
	opusEnc, err := gopus.NewEncoder(sampling, channels, gopus.Audio)
	if err != nil {
		return err
	}

	for {
		rec, ok := <-pcm
		if !ok {
			log.Debug("pcm closed")
			return nil
		}

		opus, err := opusEnc.Encode(rec, frameSize, maxBytes)
		if err != nil {
			return err
		}

		vcChan <- opus
	}
}
