package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"dbot/pkg/config"
	minIO "dbot/pkg/minio"

	"github.com/minio/minio-go/v7"
)

func main() {
	ctx := context.Background()
	config.Load()

	config.MINIO_HOST = "store-api.dodupy.dev"
	config.MINIO_ACCESS_KEY_ID = ""
	config.MINIO_SECRET_ACCESS_KEY = ""
	config.MINIO_DBOT_BUCKET_NAME = "dbot"
	config.ENV = config.Prod

	mini, err := minIO.NewMinioStore(ctx)
	if err != nil {
		panic(err)
	}

	// get all sounds and save them to config.BACKUP_DIR
	list := mini.ListObjects(ctx, config.MINIO_DBOT_BUCKET_NAME, minio.ListObjectsOptions{
		Prefix:    "492318912881491981/sounds/",
		Recursive: true,
	})

	for v := range list {
		if v.Err != nil {
			panic(v.Err)
		}
		if v.Key == "" {
			continue
		}

		file, err := mini.GetObject(ctx, config.MINIO_DBOT_BUCKET_NAME, v.Key, minio.GetObjectOptions{})
		if err != nil {
			panic(err)
		}
		defer file.Close()

		name := filepath.Base(v.Key)
		path := filepath.Join("/attached-storage/server/static/bot/", filepath.Dir(v.Key))
		err = os.MkdirAll(path, 0o755)
		if err != nil {
			panic(err)
		}
		f, err := os.Create(filepath.Join(path, name))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		io.Copy(f, file)

		fmt.Println("copied: ", v.Key)
	}
}
