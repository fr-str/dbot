package dbot

import (
	"dbot/pkg/config"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

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
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "dupa",
				Description: "dupa",
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

type cmdHandler = func(*discordgo.Session, *discordgo.InteractionCreate) error

func (d *DBot) CommandHandlers() map[string]cmdHandler {
	return map[string]cmdHandler{
		"play": d.handlePlay,
	}
}
