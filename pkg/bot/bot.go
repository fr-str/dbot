package dbot

import (
	"fmt"
	"io"
	"os/exec"

	"dbot/pkg/config"
	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
	"github.com/pion/opus/pkg/oggreader"
)

type DBot struct {
	*discordgo.Session
	ytdlp.YTDLP
}

func Start(d *discordgo.Session) {
	b := DBot{Session: d}

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
	var options struct {
		Link string `opt:"link"`
		Dupa int    `opt:"dupa"`
	}

	UnmarshalForm(i.ApplicationCommandData().Options, &options)

	channel, err := d.getUserVC(s, i.GuildID, i.Member.User.ID)
	if err != nil {
		return err
	}

	vc, err := s.ChannelVoiceJoin(i.GuildID, channel.ChannelID, false, false)
	if err != nil {
		return err
	}

	meta, err := d.DownloadAudio(options.Link)
	if err != nil {
		return err
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Playing %s", meta.Title),
		},
	})

	vc.Speaking(true)
	defer vc.Speaking(false)
	err = playFromFile(meta.Filepath, vc.OpusSend)
	if err != nil {
		return err
	}

	return nil
}

func playFromFile(fileName string, vcChan chan<- []byte) error {
	log.Debug("playFromFile", log.String("name", fileName))
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-i", fileName,
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
		return err
	}

	log.Info("playFromFile", log.String("cmd", cmd.String()))
	err = cmd.Start()
	if err != nil {
		return err
	}

	reader, _, err := oggreader.NewWith(stdout)
	if err != nil {
		return err
	}
	for {
		page, _, err := reader.ParseNextPage()
		if err != nil {
			if err != io.EOF {
				log.Error("failed to parse page", log.Err(err))
			}
			break
		}

		for _, frame := range page {
			vcChan <- frame
		}
	}

	// Wait for FFmpeg to finish
	return cmd.Wait()
}
