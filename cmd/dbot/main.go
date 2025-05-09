package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"dbot/pkg/api"
	dbot "dbot/pkg/bot"
	"dbot/pkg/config"
	"dbot/pkg/db"
	"dbot/pkg/store"
	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
)

func main() {
	config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ytdlp.StartUpdater(ctx)

	db, err := db.ConnectStore(ctx, "./test.db", "")
	if err != nil {
		panic(err)
	}

	bot(ctx, db)
}

func bot(ctx context.Context, db *store.Queries) {
	dg, err := discordgo.New(fmt.Sprintf("Bot %s", config.TOKEN))
	if err != nil {
		panic(err)
	}

	d := dbot.Start(ctx, dg, db)
	api.StartServer(d)

	<-ctx.Done()
	dg.Close()
	time.Sleep(time.Second)
}
