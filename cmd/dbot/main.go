package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	dbot "dbot/pkg/bot"
	"dbot/pkg/config"
	"dbot/pkg/db"
	"dbot/pkg/minio"
	"dbot/pkg/store"
	schema "dbot/sql"

	"github.com/bwmarrin/discordgo"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := db.ConnectStore(ctx, "./test.db", schema.DBSchema)
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

	dbot.Start(ctx, dg, db, minClient)

	<-ctx.Done()
	dg.Close()
}
