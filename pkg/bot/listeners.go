package dbot

import (
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"dbot/pkg/logic"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

func (d *DBot) RegisterEventListiners() {
	cmdHandlers := d.CommandHandlers()
	d.AddHandler(d.commands(cmdHandlers))
	d.AddHandler(d.Ready)
	d.AddHandler(d.messages)
}

func (d *DBot) commands(cmdHandlers map[string]func(*discordgo.InteractionCreate) error) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := cmdHandlers[i.ApplicationCommandData().Name]; ok {
			err := h(i)
			if err != nil {
				log.Error(err.Error())

				d.message(channelMessage{
					chid:    i.ChannelID,
					content: err.Error(),
				})
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

	sounds, err := d.Store.SelectSounds(d.Ctx, m.GuildID)
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
	sound, err := logic.FindSound(d.Store, m.Content, m.GuildID)
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

	bucket, key, found := strings.Cut(sound.Url, ",")
	if !found {
		log.Warn(fmt.Sprintf("could not find separator in link '%s'", sound.Url))
		return
	}
	url, err := d.MinIO.PresignedGetObject(d.Ctx, bucket, key, 5*time.Hour, url.Values{})
	if err != nil {
		log.Error(err.Error())
		return
	}

	log.Trace("[dupa]", log.Any("o.String()", url.String()))
	d.MusicPlayer.PlaySound(url.String())
}
