package minio

import (
	"context"
	"strings"

	"dbot/pkg/config"

	"github.com/fr-str/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Minio struct {
	*minio.Client
}

func NewMinioStore(ctx context.Context) (Minio, error) {
	ret := Minio{}
	host := config.MINIO_HOST
	accessKeyID := config.MINIO_ACCESS_KEY_ID
	secretKey := config.MINIO_SECRET_ACCESS_KEY

	minioClient, err := minio.New(host, &minio.Options{
		Creds: credentials.NewStaticV4(accessKeyID, secretKey, ""),
		// Secure: true,
	})
	if err != nil {
		return ret, err
	}
	ret.Client = minioClient

	err = ret.createDefaultBucket(ctx)
	if err != nil {
		log.Error(err.Error())
	}

	return ret, nil
}

func (m *Minio) createDefaultBucket(ctx context.Context) error {
	bucketName := config.MINIO_DBOT_BUCKET_NAME
	err := m.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: "any"})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := m.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Trace("bucket exists", log.String("name", bucketName))
			return nil
		}
		return err
	}

	log.Info("Successfully created", log.String("name", bucketName))

	return nil
}

// creates folder for each / in folder name
// example: name=dupa/123/dupa
// will create same structure in config.MINIO_DBOT_BUCKET
func (m Minio) CreateFolderStructure(ctx context.Context, name string) error {
	log.Trace("CreateFolderStructure", log.String("name", name))
	currentPath := ""
	for _, folder := range strings.Split(name, "/") {
		currentPath += folder + "/"
		// _,err := m.StatObject(ctx,config.MINIO_DBOT_BUCKET_NAME,currentPath,minio.StatObjectOptions{})
		// if err == nil {
		// 	continue
		// }

		log.Trace("CreateFolderStructure", log.Any("currentPath", currentPath))
		_, err := m.PutObject(ctx, config.MINIO_DBOT_BUCKET_NAME, currentPath, nil, 0, minio.PutObjectOptions{})
		if err != nil {
			return err
		}

	}
	return nil
}
