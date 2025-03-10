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
	var opts struct {
		Link string                       `opt:"link"`
		Att  *discordgo.MessageAttachment `opt:"file"`
	}

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &opts)
	if err != nil {
		return dbotErr("failed to parse args: %w", err)
	}

	resolved := i.ApplicationCommandData().Resolved
	if resolved == nil && len(opts.Link) == 0 {
		return errors.New("you need to provide link or attachment")
	}

	url := opts.Link
	if resolved != nil && len(resolved.Attachments) != 0 {
		opts.Att = resolved.Attachments[opts.Att.ID]
		url = opts.Att.URL
	}

	err = d.play(i.GuildID, i.Member.User.ID, url)
	if err != nil {
		return err
	}

	err = d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("added %s", url),
		},
	})
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

	d.MusicPlayer = player.NewPlayer(d._c)
	return nil
}

func (d *DBot) handleMapChannel(i *discordgo.InteractionCreate) error {
	var opts struct {
		Type    string             `opt:"type"`
		Channel *discordgo.Channel `opt:"channel"`
	}

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &opts)
	if err != nil {
		return dbotErr("failed to parse args: %w", err)
	}

	ch, err := d.mapChannel(store.MapChannelParams{
		Gid:    i.GuildID,
		Chid:   opts.Channel.ID,
		ChName: opts.Channel.Name,
		Type:   opts.Type,
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
	var opts SaveSoundParams

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &opts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal options: %w", err)
	}
	log.Debug("handleSounds", log.JSON(opts))

	opts.Link = strings.TrimSpace(opts.Link)
	// if len(options.Aliases) == 0 {
	// 	return errors.New("you need to provide aliases and a link or attachment")
	// }

	resolved := i.ApplicationCommandData().Resolved
	if resolved == nil && len(opts.Link) == 0 {
		return errors.New("you need to provide link or attachment")
	}

	if resolved != nil && len(resolved.Attachments) != 0 {
		opts.Att = resolved.Attachments[opts.Att.ID]
	}

	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "stuff",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	opts.GID = i.GuildID
	err = d.SaveSound(opts)
	if err != nil {
		return err
	}

	return nil
}

func (d *DBot) handleToMP4(i *discordgo.InteractionCreate) error {
	var opts struct {
		Link string                       `opt:"link"`
		Att  *discordgo.MessageAttachment `opt:"file"`
	}
	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &opts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal options: %w", err)
	}

	url := opts.Link
	resolved := i.ApplicationCommandData().Resolved
	if resolved != nil && len(resolved.Attachments) != 0 {
		opts.Att = resolved.Attachments[opts.Att.ID]
		url = opts.Att.URL
	}

	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Working on it, might take some time...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	log.Trace("handleToMP4", log.String("url", url))
	f, err := d.downloadAsMP4(url)
	if err != nil {
		return err
	}
	defer f.body.Close()

	_, err = d.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Files: []*discordgo.File{
			{
				Name:        "dupa.mp4",
				ContentType: "video/mp4",
				Reader:      f.body,
			},
		},
	})

	return err
}

func (d *DBot) savePlaylist(i *discordgo.InteractionCreate) error {
	var opts struct {
		Name string `opt:"name"`
		Link string `opt:"yt-link"`
	}

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &opts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal options: %w", err)
	}

	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Working on it, might take some time...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	return d.savePlaylistFromYT(opts.Name, opts.Link, i.GuildID)
}

func (d *DBot) playPlaylistFromDB(i *discordgo.InteractionCreate) error {
	if isAutocompleteInteraction(i) {
		return d.autocompleteForPlayPlaylistFromDB(i)
	}
	var opts struct {
		Name string `opt:"name"`
	}

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &opts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal options: %w", err)
	}

	err = d.connectVoice(i.GuildID, i.Member.User.ID)
	if err != nil {
		return err
	}
	err = d.loadPlaylistFromDB(opts.Name, i.GuildID)
	if err != nil {
		return err
	}

	return d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("loading %s", opts.Name),
		},
	})
}

func (d *DBot) autocompleteForPlayPlaylistFromDB(i *discordgo.InteractionCreate) error {
	names, err := d.Store.PlaylistNames(d.Ctx, i.GuildID)
	if err != nil {
		log.Error("failed getting playlist names", log.Err(err))
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(names))
	for _, name := range names {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: name, Value: name})
	}

	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	return nil
}

func isAutocompleteInteraction(i *discordgo.InteractionCreate) bool {
	return i.Type == discordgo.InteractionApplicationCommandAutocomplete
}
