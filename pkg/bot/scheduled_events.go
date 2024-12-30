package dbot

import (
	"errors"

	"dbot/pkg/config"

	"github.com/fr-str/log"
	"github.com/go-co-op/gocron/v2"
)

func (d *DBot) StartScheduler() {
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Error(err.Error())
		return
	}

	// papaj
	j, err := s.NewJob(
		gocron.CronJob("37 21 * * *", false),
		gocron.NewTask(d.papaj),
	)
	if err != nil {
		log.Error(err.Error())
	}
	log.Trace(j.Name())

	j, err = s.NewJob(
		gocron.CronJob("0 16-23/1 * * 3", false),
		gocron.NewTask(d.środowaNoc),
	)
	if err != nil {
		log.Error(err.Error())
	}

	log.Trace(j.Name())
	s.Start()
}

func (d *DBot) papaj() {
	if d.MusicPlayer.VC == nil {
		err := d.findVoiceChannel()
		if err != nil {
			log.Error(err.Error())
			return
		}
	}
	d.MusicPlayer.PlaySound("https://static.dodupy.dev/bot/soundboard/papaj.mp4")
}

func (d *DBot) środowaNoc() {
	if d.MusicPlayer.VC == nil {
		err := d.findVoiceChannel()
		if err != nil {
			log.Error(err.Error())
			return
		}
	}
	d.MusicPlayer.PlaySound("https://static.dodupy.dev/bot/soundboard/srodowanoc.mp4")
}

// only used for scheduled events
func (d *DBot) findVoiceChannel() error {
	g, _ := d.State.Guild(config.GUILD_ID)
	if g == nil {
		return errors.New("dupa")
	}

	for _, v := range g.VoiceStates {
		vc, err := d.ChannelVoiceJoin(g.ID, v.ChannelID, false, false)
		if err != nil {
			return err
		}

		d.MusicPlayer.VC = vc
		break
	}
	return nil
}
