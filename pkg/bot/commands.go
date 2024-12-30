package dbot

import (
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
				Required:    true,
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

type cmdHandler = func(*discordgo.InteractionCreate) error

func (d *DBot) CommandHandlers() map[string]cmdHandler {
	return map[string]cmdHandler{
		"play":            d.handlePlay,
		"wypierdalaj":     d.handleWypierdalaj,
		"set-bot-channel": d.mapChannel,
		"pause-play":      d.handlePause,
	}
}
