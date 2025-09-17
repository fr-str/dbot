package dbot

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"

	"dbot/pkg/ffmpeg"
	"dbot/pkg/store"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

func (d *DBot) handlePlay(ctx context.Context, i *discordgo.InteractionCreate) error {
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
			Flags:   discordgo.MessageFlagsSuppressEmbeds,
		},
	})
	return err
}

func (d *DBot) handleWypierdalaj(ctx context.Context, i *discordgo.InteractionCreate) error {
	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "sadge",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	d.wypierdalajZVC(i.GuildID)
	return nil
}

func (d *DBot) handleMapChannel(ctx context.Context, i *discordgo.InteractionCreate) error {
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

func (d *DBot) handlePause(ctx context.Context, i *discordgo.InteractionCreate) error {
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

func (d *DBot) handleSound(ctx context.Context, i *discordgo.InteractionCreate) error {
	var opts SaveSoundParams

	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &opts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal options: %w", err)
	}
	log.Debug("handleSounds", log.JSON(opts))

	opts.Link = strings.TrimSpace(opts.Link)

	resolved := i.ApplicationCommandData().Resolved
	if resolved == nil && len(opts.Link) == 0 {
		return errors.New("you need to provide link or attachment")
	}

	if resolved != nil && len(resolved.Attachments) != 0 {
		opts.Link = resolved.Attachments[opts.Att.ID].URL
	}

	d.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Saving sound '%s'", opts.Link),
		},
	})

	opts.GID = i.GuildID
	err = d.SaveSound(ctx, opts)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Saved sound '%s', '%s'", strings.Split(opts.Aliases, ",")[0], opts.Link)
	d.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &msg,
	})
	return nil
}

func (d *DBot) handleToMP4(ctx context.Context, i *discordgo.InteractionCreate) error {
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

	info, err := d.DownloadVideo(ctx, url)
	if err != nil {
		return fmt.Errorf("failed downloading video: %w", err)
	}

	f, err := ffmpeg.ConvertToMP4(ctx, info.Filepath)
	if err != nil {
		return fmt.Errorf("failed converting to mp4: %w", err)
	}

	stat, err := f.Stat()
	log.Trace("handleToMP4", log.String("file", f.Name()), log.Int("size_KB", stat.Size()>>10))
	if err != nil {
		return fmt.Errorf("failed getting file size: %w", err)
	}

	if stat.Size() > 10*1_000_000 {
		log.Info("failed converting to mp4, trying to convert to discord mp4")
		msg := "file is too big, reducing bitrate and resolution and running duble pass"
		d.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})

		f, err = ffmpeg.ToDiscordMP4(ctx, info.Filepath)
		if err != nil {
			return fmt.Errorf("failed converting to mp4: %w", err)
		}
	}

	hook, err := d.GetWebHook(ctx, i.ChannelID, DbotHook, "")
	if err != nil {
		return fmt.Errorf("getWebhook: %w", err)
	}

	_, err = d.WebhookExecute(hook.ID, hook.Token, false, &discordgo.WebhookParams{
		Username:  cmp.Or(i.Member.User.Username, i.Member.User.GlobalName),
		AvatarURL: i.Member.User.AvatarURL(""),
		Files: []*discordgo.File{
			{
				Name:        "dupa.mp4",
				ContentType: "video/mp4",
				Reader:      f,
			},
		},
		Flags: discordgo.MessageFlagsSuppressEmbeds,
	})

	return err
}

func (d *DBot) savePlaylist(ctx context.Context, i *discordgo.InteractionCreate) error {
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

func (d *DBot) playPlaylistFromDB(ctx context.Context, i *discordgo.InteractionCreate) error {
	if isAutocompleteInteraction(i) {
		return d.autocompleteForPlayPlaylist(i)
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

func (d *DBot) autocompleteForPlayPlaylist(i *discordgo.InteractionCreate) error {
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
