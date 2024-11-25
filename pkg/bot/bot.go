package dbot

import (
	"fmt"

	"dbot/pkg/config"
	"dbot/pkg/player"
	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

type DBot struct {
	*discordgo.Session
	ytdlp.YTDLP
	Player *player.Player
}

func dbotErr(msg string, vars ...any) error {
	return fmt.Errorf("dbot: "+msg+": ", vars...)
}

func Start(d *discordgo.Session) {
	b := DBot{Session: d, Player: player.NewPlayer()}

	d.AddHandler(b.Ready)
	err := d.Open()
	if err != nil {
		panic(err)
	}

	cmdHandlers := b.CommandHandlers()
	d.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := cmdHandlers[i.ApplicationCommandData().Name]; ok {
			err := h(i)
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

	go func() {
		for err := range b.Player.ErrChan {
			log.Error(err.Error())
		}
	}()
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

func (d *DBot) handlePlay(i *discordgo.InteractionCreate) error {
	var options struct {
		Link string `opt:"link"`
		Dupa int    `opt:"dupa"`
	}

	err := UnmarshalOptions(i.ApplicationCommandData().Options, &options)
	if err != nil {
		return dbotErr("failed to parse args: %w", err)
	}

	channel, err := d.getUserVC(d.Session, i.GuildID, i.Member.User.ID)
	if err != nil {
		return err
	}
	if channel == nil {
		return dbotErr("user not in channel")
	}

	vc, err := d.ChannelVoiceJoin(i.GuildID, channel.ChannelID, false, false)
	if err != nil {
		return err
	}

	if d.Player.VC == nil {
		d.Player.VC = vc
	}

	err = d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "added",
		},
	})

	d.Player.Add(options.Link)

	return nil
}

func (d *DBot) handleWypierdalaj(i *discordgo.InteractionCreate) error {
	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "sadge",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if d.Player.VC != nil {
		return d.Player.VC.Disconnect()
	}
	return nil
}
