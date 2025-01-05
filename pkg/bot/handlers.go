package dbot

import (
	"errors"
	"fmt"
	"strings"

	"dbot/pkg/player"
	"dbot/pkg/store"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

func (d *DBot) handlePlay(i *discordgo.InteractionCreate) error {
	var options struct {
		Link string `opt:"link"`
	}

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &options)
	if err != nil {
		return dbotErr("failed to parse args: %w", err)
	}

	err = d.connectVoice(i.GuildID, i.User.ID)
	if err != nil {
		return err
	}

	err = d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("added %s", options.Link),
		},
	})

	d.MusicPlayer.Add(options.Link)

	return err
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
		err := d.MusicPlayer.VC.Disconnect()
		if err != nil {
			d.MusicPlayer.ErrChan <- player.Err{
				GID: i.GuildID,
				Err: err,
			}
		}
	}

	d.MusicPlayer = player.NewPlayer()
	return nil
}

func (d *DBot) handleMapChannel(i *discordgo.InteractionCreate) error {
	var options struct {
		Type    string             `opt:"type"`
		Channel *discordgo.Channel `opt:"channel"`
	}

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &options)
	if err != nil {
		return dbotErr("failed to parse args: %w", err)
	}

	ch, err := d.mapChannel(store.MapChannelParams{
		Gid:    i.GuildID,
		Chid:   options.Channel.ID,
		ChName: options.Channel.Name,
		Type:   options.Type,
	})
	if err != nil {
		return err
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

func (d *DBot) handlePause(i *discordgo.InteractionCreate) error {
	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("paused %v", !d.MusicPlayer.Playing.Load()),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	d.MusicPlayer.PlayPause()
	return nil
}

func (d *DBot) handleSound(i *discordgo.InteractionCreate) error {
	var params SaveSoundParams

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &params)
	if err != nil {
		return fmt.Errorf("failed to unmarshal options: %w", err)
	}
	log.Debug("handleSounds", log.JSON(params))

	params.Link = strings.TrimSpace(params.Link)
	params.Aliases = normalizeReplacer.Replace(params.Aliases)
	// if len(options.Aliases) == 0 {
	// 	return errors.New("you need to provide aliases and a link or attachment")
	// }

	resolved := i.ApplicationCommandData().Resolved
	if resolved == nil && len(params.Link) == 0 {
		return errors.New("you need to provide link or attachment")
	}

	if resolved != nil && len(resolved.Attachments) != 0 {
		params.Att = resolved.Attachments[params.Att.ID]
	}

	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "stuff",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	params.GID = i.GuildID
	err = d.SaveSound(params)
	if err != nil {
		return err
	}

	return nil
}
