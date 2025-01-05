package minio

import (
	"context"

	"dbot/pkg/config"

	"github.com/fr-str/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinioStore(ctx context.Context) (*minio.Client, error) {
	host := config.MINIO_HOST
	accessKeyID := config.MINIO_ACCESS_KEY_ID
	secretKey := config.MINIO_SECRET_ACCESS_KEY

	minioClient, err := minio.New(host, &minio.Options{
		Creds: credentials.NewStaticV4(accessKeyID, secretKey, ""),
	})
	if err != nil {
		return nil, err
	}

	err = createDefaultBucket(ctx, minioClient)
	if err != nil {
		log.Error(err.Error())
	}

	return minioClient, nil
}

func createDefaultBucket(ctx context.Context, mIO *minio.Client) error {
	bucketName := config.MINIO_DBOT_BUCKET_NAME

	err := mIO.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: "any"})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := mIO.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Trace("bucket exists", log.String("name", bucketName))
			return nil
		}
		return err
	}

	log.Info("Successfully created", log.String("name", bucketName))
	return nil
}
