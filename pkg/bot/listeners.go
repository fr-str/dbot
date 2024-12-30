package dbot

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

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
	soundAll(d, m)
	// testPlay(d, m)
}

func soundAll(d *DBot, m *discordgo.MessageCreate) {
	m.Content = strings.ToLower(strings.ReplaceAll(m.Content, " ", ""))
	if m.Content != "sound-all" {
		return
	}

	log.Debug("sound all", log.String("msg", m.Content))

	sounds, err := d.store.SelectSounds(d.ctx, m.GuildID)
	if err != nil {
		log.Error(err.Error())
		return
	}

	rand.Shuffle(len(sounds), func(i, j int) {
		sounds[i], sounds[j] = sounds[j], sounds[i]
	})

	err = d.connectVoice(m.GuildID, m.Author.ID)
	if err != nil {
		log.Error(err.Error())
		return
	}

	for _, s := range sounds {
		d.MusicPlayer.PlaySound(s.Url)
	}
}

func testPlay(d *DBot, m *discordgo.MessageCreate) {
	err := d.connectVoice(m.GuildID, m.Author.ID)
	if err != nil {
		log.Error(err.Error())
		return
	}

	d.MusicPlayer.Add(m.Content)
}

func (d *DBot) connectVoice(gID, uID string) error {
	// skip if we are already connected
	if d.MusicPlayer.VC != nil {
		return nil
	}

	channel, err := d.getUserVC(d.Session, gID, uID)
	if err != nil {
		return fmt.Errorf("failed to find User VC: %w", err)
	}

	vc, err := d.ChannelVoiceJoin(gID, channel.ChannelID, false, false)
	if err != nil {
		log.Error("failed to join VC", log.Err(err))
		return fmt.Errorf("failed to join VC: %w", err)
	}

	d.MusicPlayer.VC = vc

	return nil
}

func isKnownSound(d *DBot, m *discordgo.MessageCreate) {
	log.Debug("isKnownSound", log.String("msg", m.Content))
	m.Content = strings.ToLower(strings.ReplaceAll(m.Content, " ", ""))
	sound, err := logic.FindSound(d.store, m.Content, m.GuildID)
	if err != nil {
		if !errors.Is(err, logic.ErrSoundNotFound) {
			log.Info("failed to find sound", log.Err(err))
		}
		return
	}

	err = d.connectVoice(m.GuildID, m.Author.ID)
	if err != nil {
		log.Error(err.Error())
		return
	}

	d.MusicPlayer.PlaySound(sound.Url)
}
