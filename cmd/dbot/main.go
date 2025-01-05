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
	"dbot/pkg/minio"
	"dbot/pkg/store"

	"github.com/bwmarrin/discordgo"
	miniocli "github.com/minio/minio-go/v7"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := db.Connect(ctx, "./test.db", schema.Schema)
	if err != nil {
		panic(err)
	}

	minClien, err := minio.NewMinioStore(ctx)
	if err != nil {
		panic(err)
	}

	bot(ctx, db, minClien)
}

func bot(ctx context.Context, db *store.Queries, minClient *miniocli.Client) {
	dg, err := discordgo.New(fmt.Sprintf("Bot %s", config.TOKEN))
	if err != nil {
		panic(err)
	}

	dbot.Start(ctx, dg, db, minClient)
	// name := ""
	// for f := range minClient.ListObjects(ctx, config.MINIO_DBOT_BUCKET_NAME, miniocli.ListObjectsOptions{}) {
	// 	log.Trace("[dupa]", log.JSON(f))
	// 	name = f.Key
	// }
	//
	// o, err := minClient.GetObject(ctx, config.MINIO_DBOT_BUCKET_NAME, name, miniocli.GetObjectOptions{})
	// if err != nil {
	// 	log.Error(err.Error())
	// }
	// ob, _ := o.Stat()
	// log.Trace("[dupa]", log.JSON(ob))

	<-ctx.Done()
	dg.Close()
}
