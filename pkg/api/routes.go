package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	dbot "dbot/pkg/bot"

	"github.com/fr-str/log"
)

func StartServer(d *dbot.DBot) {
	http.HandleFunc("/api/dst/links", func(w http.ResponseWriter, r *http.Request) {
		links := make(map[string]string)
		err := json.NewDecoder(r.Body).Decode(&links)
		if err != nil {
			log.Error(err.Error())
			return
		}
		if len(links) == 0 {
			log.Error("no links")
			return
		}
		log.Trace("links", log.JSON(links))

		msg := "New	links:\n"
		for k, v := range links {
			msg += fmt.Sprintf("[URL](%s) Points: %s\n", k, v)
		}

		_, err = d.ChannelMessageSend("983810486627876924", msg)
		if err != nil {
			log.Error(err.Error())
		}

		w.WriteHeader(http.StatusOK)
	})
	go func() {
		log.Info("starting server: http://localhost:58008")
		err := http.ListenAndServe(":58008", nil)
		if err != nil {
			log.Error(err.Error())
		}
	}()
}
