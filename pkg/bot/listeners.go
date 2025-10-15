package dbot

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"dbot/pkg/config"
	"dbot/pkg/ffmpeg"
	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

func (d *DBot) RegisterEventListiners() {
	cmdHandlers := d.CommandHandlers()
	d.AddHandler(d.commands(cmdHandlers))
	d.AddHandler(d.Ready)

	d.AddHandler(d.messagesListener)
	d.AddHandler(d.messagesEditListener)
	d.AddHandler(d.onUserVoiceStateChange)
}

func createContextTmpDir(ctx context.Context) context.Context {
	dir, err := os.MkdirTemp(config.TMP_PATH, strconv.Itoa(int(time.Now().UnixNano())))
	if err != nil {
		log.Error("dupa", log.Err(err))
	}

	ctx = context.WithValue(ctx, config.DirKey, dir)
	go func() {
		<-ctx.Done()
		os.RemoveAll(dir)
	}()
	return ctx
}

func (d *DBot) commands(cmdHandlers map[string]cmdHandler) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := cmdHandlers[i.ApplicationCommandData().Name]; ok {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx = createContextTmpDir(ctx)

			err := h(ctx, i)
			if err != nil {
				log.Error("uuu duuuuuuuuuuuuuuuuuupa", log.Err(err), log.String("cmd", i.ApplicationCommandData().Name), log.String("err_type", fmt.Sprintf("%T", err)))

				msg := err.Error()
				if errors.Is(err, ytdlp.ErrFailedToDownload) {
					msg = "could not download video"
				}
				if len(msg) > 2000 {
					msg = msg[:2000-1]
				}
				_, err := d.ChannelMessageSend(i.ChannelID, msg)
				if err != nil {
					log.Error("response failed", log.Err(err))
				}
			}
		}
	}
}

func (d *DBot) messagesEditListener(_ *discordgo.Session, m *discordgo.MessageUpdate) {
	if m.Author.Bot {
		return
	}

	err := updateBackupMessage(d, m.Message)
	if err != nil {
		log.Error("updateBackupMessage", log.Err(err))
	}
}

func (d *DBot) messagesListener(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}
	// err := backupMessage(d, m.Message)
	// if err != nil {
	// 	log.Error("backupMessage", log.Err(err))
	// }

	isKnownSound(d, m)
	soundAll(d, m)
	// transcodeToh264(d, m)
}

func transcodeToh264(d *DBot, m *discordgo.MessageCreate) {
	if len(m.Attachments) == 0 {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctx = createContextTmpDir(ctx)
	defer cancel()
	badCodecs := []string{
		"hevc",
		"av1",
		"h265",
	}

	for _, att := range m.Attachments {
		if !strings.Contains(att.ContentType, "video") {
			continue
		}
		meta, err := d.DownloadVideo(ctx, att.URL)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		videoInfo, err := ffmpeg.Probe(meta.Filepath)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		for i := range videoInfo.Streams {
			s := &videoInfo.Streams[i]
			if s.CodecType != "video" {
				continue
			}

			if !slices.Contains(badCodecs, s.CodecName) {
				continue
			}

			d.MessageReactionAdd(m.ChannelID, m.ID, "bosy:1220157705273086002")

			// transcode and upload
			f, err := ffmpeg.ConvertToMP4(ctx, meta.Filepath)
			if err != nil {
				log.Error("transcodeAndReupload", log.Err(err))
				return
			}
			defer f.Close()
			defer os.Remove(f.Name())
			msg, err := d.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
				Files: []*discordgo.File{
					{
						Name:        "dupa.mp4",
						ContentType: "video/mp4",
						Reader:      f,
					},
				},
			})
			if err != nil {
				log.Error("transcodeAndReupload", log.Err(err))
				return
			}
			d.MessageReactionAdd(msg.ChannelID, msg.ID, "skipper:1330256451331031162")
		}
	}
}

func soundAll(d *DBot, m *discordgo.MessageCreate) {
	normalized := normalize(m.Content)
	if normalized != "sound-all" && normalized != "event-all" {
		return
	}

	log.Debug("sound all", log.String("msg", normalized))

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

func isKnownSound(d *DBot, m *discordgo.MessageCreate) {
	msg := m.Content
	normalized := normalize(m.Content)
	log.Debug("isKnownSound", log.String("msg", msg), log.String("normalized", normalized))
	sound, err := findSound(d.Store, normalized, m.GuildID)
	if err != nil {
		if !errors.Is(err, ErrSoundNotFound) {
			log.Info("failed to find sound", log.Err(err))
		}
		return
	}

	err = d.connectVoice(m.GuildID, m.Author.ID)
	if err != nil {
		log.Error(err.Error())
		return
	}

	for _, s := range sound {
		if strings.Contains(s.Url, "riotgames") {
			continue
		}
		log.Trace("isKnownSound", log.Any("sound.Url", s.Url), log.Any("name", s.Aliases))
		d.MusicPlayer.PlaySound(s.Url)
	}
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

	botChanID := d.MusicPlayer.VCID
	log.Trace("onUserVoiceStateChange", log.Any("botChanID", botChanID))
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
