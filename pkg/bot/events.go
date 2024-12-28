package dbot

import (
	"dbot/pkg/logic"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

func (d *DBot) RegisterEventListiners() {
	cmdHandlers := d.CommandHandlers()
	d.AddHandler(commands(cmdHandlers))
	d.AddHandler(d.Ready)
	d.AddHandler(d.messages)
}

func commands(cmdHandlers map[string]func(*discordgo.InteractionCreate) error) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := cmdHandlers[i.ApplicationCommandData().Name]; ok {
			err := h(i)
			if err != nil {
				log.Error(err.Error())
			}
		}
	}
}

func (d *DBot) messages(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	isKnownSound(d, m)
	// testPlay(d, m)
}

func testPlay(d *DBot, m *discordgo.MessageCreate) {
	channel, err := d.getUserVC(d.Session, m.GuildID, m.Author.ID)
	if err != nil {
		log.Error(err.Error())
		return
	}

	vc, err := d.ChannelVoiceJoin(m.GuildID, channel.ChannelID, false, false)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if d.MusicPlayer.VC == nil {
		d.MusicPlayer.VC = vc
	}

	d.MusicPlayer.Add(m.Content)
}

func isKnownSound(d *DBot, m *discordgo.MessageCreate) {
	sound, err := logic.FindSound(d.store, m.Content, m.GuildID)
	if err != nil {
		log.Error("failed to find sound", log.Err(err))
		return
	}

	channel, err := d.getUserVC(d.Session, m.GuildID, m.Author.ID)
	if err != nil {
		log.Error("failed to find User VC", log.Err(err))
		return
	}

	vc, err := d.ChannelVoiceJoin(m.GuildID, channel.ChannelID, false, false)
	if err != nil {
		log.Error("failed to join VC", log.Err(err))
		return
	}

	if d.MusicPlayer.VC == nil {
		d.MusicPlayer.VC = vc
	}

	d.MusicPlayer.PlaySound(sound.Url)
}
