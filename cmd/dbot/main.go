package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"dbot/pkg/api"
	dbot "dbot/pkg/bot"
	"dbot/pkg/config"
	"dbot/pkg/db"
	"dbot/pkg/minio"
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

	minClien, err := minio.NewMinioStore(ctx)
	if err != nil {
		panic(err)
	}

	bot(ctx, db, minClien)
}

func bot(ctx context.Context, db *store.Queries, minClient minio.Minio) {
	dg, err := discordgo.New(fmt.Sprintf("Bot %s", config.TOKEN))
	if err != nil {
		panic(err)
	}

	d := dbot.Start(ctx, dg, db, minClient)
	api.StartServer(d)

	<-ctx.Done()
	dg.Close()
}
