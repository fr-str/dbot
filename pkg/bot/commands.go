package dbot

import (
	"context"

	"dbot/pkg/config"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

// descryptions are requeired by Discord
var cmds = []*discordgo.ApplicationCommand{
	{
		Name:        "play",
		Description: "play audio from source",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "link",
				Description: "dupa",
			},
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        "file",
				Description: "file",
			},
		},
	},
	{
		Name:        "pause-play",
		Description: "play pause",
	},
	{
		Name:        "wypierdalaj",
		Description: "bot wpierdala",
	},
	{
		Name:        "to-mp4",
		Description: "Attempts to transcode video to format compatible with discord.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "link",
				Description: "link to audio",
			},
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        "file",
				Description: "file",
			},
		},
	},
	{
		Name:        "sound",
		Description: "add new sound",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "aliases",
				Description: "triggers for sound",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "link",
				Description: "link to audio",
			},
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        "file",
				Description: "file",
			},
		},
	},
	{
		Name:        "set-bot-channel",
		Description: "sets channels for bot to work in",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "type",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "channel",
				Description: "channel",
				Required:    true,
			},
		},
	},
	{
		Name:        "save-playlist",
		Description: "saves playlist and downloads all avaiable videos",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "yt-link",
				Description: "link",
				Required:    true,
			},
		},
	},
	{
		Name:        "load-playlist",
		Description: "loads saved playlist",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "name",
				Description:  "name",
				Autocomplete: true,
				Required:     true,
			},
		},
	},
}

func (d *DBot) ClearCmds(s *discordgo.Session) {
	c, err := s.ApplicationCommands(s.State.User.ID, config.GUILD_ID)
	if err != nil {
		log.Error(err.Error())
	}

	for _, v := range c {
		err := s.ApplicationCommandDelete(s.State.User.ID, config.GUILD_ID, v.ID)
		if err != nil {
			log.Error(err.Error())
		}
	}
}

type cmdHandler = func(context.Context, *discordgo.InteractionCreate) error

func (d *DBot) CommandHandlers() map[string]cmdHandler {
	return map[string]cmdHandler{
		"play":            d.handlePlay,
		"wypierdalaj":     d.handleWypierdalaj,
		"set-bot-channel": d.handleMapChannel,
		"pause-play":      d.handlePause,
		"sound":           d.handleSound,
		"to-mp4":          d.handleToMP4,
		"save-playlist":   d.savePlaylist,
		"load-playlist":   d.playPlaylistFromDB,
	}
}
