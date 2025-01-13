package dbot

import (
	"fmt"
	"time"

	"dbot/pkg/store"

	"github.com/fr-str/log"
)

func (d *DBot) interfaceLoop() {
	tic := time.NewTicker(time.Second * 3)
	var lastContent string
	var lastID string
	for range tic.C {
		if d.MusicPlayer.VC == nil {
			continue
		}
		c, err := d.Store.GetChannel(d.Ctx, store.GetChannelParams{
			Gid:  d.MusicPlayer.VC.GuildID,
			Type: "music",
		})
		if err != nil {
			continue
		}
		current := d.MusicPlayer.Current()
		if current == nil {
			continue
		}

		content := fmt.Sprintf("Playing [%s](%s)", current.Title, current.Link)
		if content == lastContent {
			continue
		}
		err = d.ChannelMessageDelete(c.Chid, lastID)
		if err != nil {
			log.Error(err.Error())
		}

		lastContent = content
		msg, err := d.ChannelMessageSend(c.Chid, content)
		if err != nil {
			log.Error(err.Error())
		}
		lastID = msg.ID

	}
}
