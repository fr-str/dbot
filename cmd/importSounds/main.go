package main

import (
	"context"
	"encoding/json"
	"os"

	schema "dbot"
	"dbot/pkg/db"
	"dbot/pkg/store"
)

func main() {
	ctx := context.Background()

	db, err := db.Connect(ctx, "./test.db", schema.Schema)
	if err != nil {
		panic(err)
	}

	for k, v := range parse() {
		_, err := db.AddSound(ctx, store.AddSoundParams{
			Gid:     "438758201916129281",
			Url:     k,
			Aliases: v,
		})
		if err != nil {
			panic(err)
		}
	}
}

func parse() map[string][]string {
	f, err := os.Open("./492318912881491981.json")
	if err != nil {
		panic(err)
	}

	var m map[string][]string
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		panic(err)
	}
	return m
}
