package dbot

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"dbot/pkg/dbg"
	"dbot/pkg/logic"
	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

func (d *DBot) RegisterEventListiners() {
	cmdHandlers := d.CommandHandlers()
	d.AddHandler(d.commands(cmdHandlers))
	d.AddHandler(d.Ready)
	d.AddHandler(d.messages)
	d.AddHandler(d.onUserVoiceStateChange)
}

func (d *DBot) commands(cmdHandlers map[string]func(*discordgo.InteractionCreate) error) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := cmdHandlers[i.ApplicationCommandData().Name]; ok {
			err := h(i)
			if err != nil {
				log.Error(err.Error())

				msg := err.Error()
				if errors.Is(err, ytdlp.ErrFailedToDownload) {
					msg = "could not download video"
				}
				_, err := d.ChannelMessageSend(i.ChannelID, msg)
				if err != nil {
					log.Error("response failed", log.Err(err))
				}
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
	if m.Content != "sound-all" && m.Content != "event-all" {
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
		dbg.Assert(len(s.Gid) > 0)
		dbg.Assert(len(s.Url) > 0)
		dbg.Assert(len(s.Aliases) > 0)
		url, err := d.getLinkFromSoundKey(s.Url)
		if err != nil {
			log.Error(err.Error())
			return
		}
		d.MusicPlayer.PlaySound(url)
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
		return fmt.Errorf("failed to join VC: %w", err)
	}

	d.MusicPlayer.VC = vc

	return nil
}

func isKnownSound(d *DBot, m *discordgo.MessageCreate) {
	msg := m.Content
	m.Content = normalize(m.Content)
	log.Debug("isKnownSound", log.String("msg", msg), log.String("normalized", m.Content))
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

	log.Trace("isKnownSound", log.Any("sound.Url", sound.Url))

	url, err := d.getLinkFromSoundKey(sound.Url)
	if err != nil {
		log.Error(err.Error())
		return
	}

	log.Trace("isKnownSound", log.Any("url", url))
	d.MusicPlayer.PlaySound(url)
}

func (d *DBot) onUserVoiceStateChange(_ *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	g, _ := d.State.Guild(vs.GuildID)
	if g == nil {
		return
	}

	log.Trace("onUserVoiceStateChange", log.Any("d.MusicPlayer.VC == nil", d.MusicPlayer.VC == nil))
	if d.MusicPlayer.VC == nil {
		return
	}

	botChanID := d.MusicPlayer.VC.ChannelID
	log.Trace("onUserVoiceStateChange", log.Any("d.MusicPlayer.VC.ChannelID", d.MusicPlayer.VC.ChannelID))
	var botChanUserCount int
	for _, v := range g.VoiceStates {
		if v.Member != nil && v.Member.User.Bot {
			continue
		}
		if v.ChannelID != botChanID {
			continue
		}

		botChanUserCount++
	}

	if botChanUserCount > 0 {
		return
	}

	d.wypierdalajZVC(vs.GuildID)
}
