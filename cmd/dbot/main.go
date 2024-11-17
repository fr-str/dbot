package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	dbot "dbot/pkg/bot"
	"dbot/pkg/config"

	"github.com/bwmarrin/discordgo"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// y := ytdlp.YTDLP{}
	// y.DownloadAudio("https://static.dodupy.dev/bot/soundboard/padasnieg.mp4")

	bot(ctx)
}

func bot(ctx context.Context) {
	dg, err := discordgo.New(fmt.Sprintf("Bot %s", config.TOKEN))
	if err != nil {
		panic(err)
	}

	dbot.Start(dg)

	<-ctx.Done()
	dg.Close()
}
