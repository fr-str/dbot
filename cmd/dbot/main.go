package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	schema "dbot"

	dbot "dbot/pkg/bot"
	"dbot/pkg/config"
	"dbot/pkg/db"
	"dbot/pkg/store"

	"github.com/bwmarrin/discordgo"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := db.Connect(ctx, "./test.db", schema.Schema)
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

	dbot.Start(ctx, dg, db)

	<-ctx.Done()
	dg.Close()
}
