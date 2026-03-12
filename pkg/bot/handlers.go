package dbot

import (
	"context"
	"errors"
	"fmt"
	"os"
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

	d.wypierdalajZVC()
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
		Link   string                       `opt:"link"`
		Att    *discordgo.MessageAttachment `opt:"file"`
		Mute   bool                         `opt:"mute"`
		Format string                       `opt:"format"`
	}
	err := UnmarshalOptions(d.Session, i.ApplicationCommandData().Options, &opts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal options: %w", err)
	}

	// Default to MP4 if no format specified
	if opts.Format == "" {
		opts.Format = "mp4"
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

	info, err := d.DownloadVideoSmall(ctx, url)
	if err != nil {
		return fmt.Errorf("failed downloading video: %w", err)
	}

	f, err := os.Open(info.Filepath)
	if err != nil {
		return fmt.Errorf("dupa")
	}
	defer f.Close()

	stat, err := f.Stat()
	log.Trace("handleToMP4", log.String("file", f.Name()), log.Int("size_KB", stat.Size()>>10))
	if err != nil {
		return fmt.Errorf("failed getting file size: %w", err)
	}

	var maxsizebytes int64 = 10 * 1_000_000
	// Always convert if GIF format is requested, or if file is too large or mute is requested for MP4
	if opts.Format == "gif" || stat.Size() > maxsizebytes || opts.Mute {
		if opts.Format == "gif" {
			attempts := []ffmpeg.GifSettings{
				{Height: 320, FPS: 15},
				{Height: 280, FPS: 12},
				{Height: 240, FPS: 10},
				{Height: 180, FPS: 8},
			}

			msg := "converting to GIF..."
			d.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
			for _, settings := range attempts {
				log.Info("converting to GIF")

				f, err = ffmpeg.ToDiscordGIF(ctx, info.Filepath, settings)
				if err != nil {
					return fmt.Errorf("failed converting to GIF: %w", err)
				}
				defer f.Close()

				info, err := f.Stat()
				if err != nil {
					return err
				}

				if info.Size() < maxsizebytes {
					log.Info(fmt.Sprintf("Success! GIF created at %dMB using Height:%d FPS:%d\n", info.Size()/1024/1024, settings.Height, settings.FPS))
					break
				}

				log.Info(fmt.Sprintf("File too large (%dMB). Retrying with lower quality...\n", info.Size()/1024/1024))
			}
		} else {
			log.Info("converting to MP4, file too big or mute requested")
			msg := "file is too big, reducing bitrate and resolution and running double pass"
			d.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})

			f, err = ffmpeg.ToDiscordMP4(ctx, info.Filepath, opts.Mute)
			if err != nil {
				return fmt.Errorf("failed converting to MP4: %w", err)
			}
			defer f.Close()
		}
	}

	hook, err := d.GetWebHook(ctx, i.ChannelID, DbotHook, "")
	if err != nil {
		return fmt.Errorf("getWebhook: %w", err)
	}

	user, _ := d.GuildMember(i.GuildID, i.Member.User.ID)
	if user == nil {
		user = &discordgo.Member{}
	}

	// Determine filename and content type based on format
	var fileName, contentType string
	if opts.Format == "gif" {
		fileName = "dupa.gif"
		contentType = "image/gif"
	} else {
		fileName = "dupa.mp4"
		contentType = "video/mp4"
	}

	_, err = d.WebhookExecute(hook.ID, hook.Token, false, &discordgo.WebhookParams{
		Username:  user.DisplayName(),
		AvatarURL: i.Member.User.AvatarURL(""),
		Files: []*discordgo.File{
			{
				Name:        fileName,
				ContentType: contentType,
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
