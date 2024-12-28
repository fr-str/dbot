package dbot

import (
	"context"
	"fmt"

	"dbot/pkg/config"
	"dbot/pkg/player"
	"dbot/pkg/store"
	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

type DBot struct {
	ctx   context.Context
	store *store.Queries

	*discordgo.Session
	ytdlp.YTDLP
	// MusicPlayer is for plaing stuff from YT and others
	MusicPlayer *player.Player

	// SoundPlayer is for soundboard
	SoundPlayer *player.Player
}

func dbotErr(msg string, vars ...any) error {
	return fmt.Errorf("dbot: "+msg+": ", vars...)
}

func Start(ctx context.Context, sess *discordgo.Session, db *store.Queries) {
	d := DBot{
		ctx:         ctx,
		Session:     sess,
		MusicPlayer: player.NewPlayer(),
		SoundPlayer: player.NewPlayer(),
		store:       db,
	}

	// Listeners must be registered befor we open connection
	d.RegisterEventListiners()

	err := sess.Open()
	if err != nil {
		panic(err)
	}

	for _, v := range cmds {
		_, err := sess.ApplicationCommandCreate(sess.State.User.ID, config.GUILD_ID, v)
		if err != nil {
			panic(err)
		}
	}

	go func() {
		for {
			var Err player.Err
			select {
			case Err = <-d.MusicPlayer.ErrChan:
			case Err = <-d.SoundPlayer.ErrChan:
			}

			log.Error(Err.Err.Error())
			ch, err := d.store.GetChannel(ctx, store.GetChannelParams{
				Gid:  Err.GID,
				Type: musicChannel,
			})
			if err != nil {
				log.Error(err.Error())
				continue
			}

			d.message(channelMessage{
				chid:    ch.Chid,
				content: Err.Err.Error(),
			})
		}
	}()
}

const (
	musicChannel = "music"
	errorChannel = "error"
	adminChannel = "admin"
)

type response struct {
	*discordgo.Interaction
	msg *discordgo.InteractionResponse
	typ string
}

// use this to send response to user intearaction
func (d *DBot) respond(response response) {
	err := d.InteractionRespond(response.Interaction, response.msg)
	if err != nil {
		log.Error("response failed", log.Err(err), log.JSON(response))
	}
}

type channelMessage struct {
	chid    string
	content string
}

// use this to send message not attached to user interaction
func (d *DBot) message(msg channelMessage) {
	_, err := d.ChannelMessageSend(msg.chid, msg.content)
	if err != nil {
		log.Error("response failed", log.Err(err), log.JSON(msg))
	}
}

func (d *DBot) Ready(s *discordgo.Session, e *discordgo.Ready) {
	log.Info("ready")
}

// returns users VoiceCHannel if user is connected to one
// errors if guild does not exist and if user is not in voice channel
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

	return nil, fmt.Errorf("user '%s' in guild '%s' is not in voice channel", uID, gID)
}

func (d *DBot) handlePlay(i *discordgo.InteractionCreate) error {
	var options struct {
		Link string `opt:"link"`
	}

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &options)
	if err != nil {
		return dbotErr("failed to parse args: %w", err)
	}

	channel, err := d.getUserVC(d.Session, i.GuildID, i.Member.User.ID)
	if err != nil {
		return err
	}

	vc, err := d.ChannelVoiceJoin(i.GuildID, channel.ChannelID, false, false)
	if err != nil {
		return err
	}

	if d.MusicPlayer.VC == nil {
		d.MusicPlayer.VC = vc
	}

	err = d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "added",
		},
	})

	d.MusicPlayer.Add(options.Link)

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

	if d.MusicPlayer.VC != nil {
		return d.MusicPlayer.VC.Disconnect()
	}

	d.MusicPlayer = player.NewPlayer()
	return nil
}

func (d *DBot) mapChannel(i *discordgo.InteractionCreate) error {
	var options struct {
		Type    string             `opt:"type"`
		Channel *discordgo.Channel `opt:"channel"`
	}

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &options)
	if err != nil {
		return dbotErr("failed to parse args: %w", err)
	}

	ch, err := d.store.MapChannel(d.ctx, store.MapChannelParams{
		Gid:    i.GuildID,
		Chid:   options.Channel.ID,
		ChName: options.Channel.Name,
		Type:   options.Type,
	})
	if err != nil {
		return dbotErr("failed to save: %w", err)
	}

	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Channel <#%s> will be used as '%s' channel", ch.Chid, ch.Type),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	return nil
}
