// Package storage handles uploading user-supplied images to S3.
package storage

import (
	"context"
	"fmt"
	"loof/internal/config"
	"mime/multipart"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// MaxImageSize is the hard limit on a single uploaded image, in bytes.
const MaxImageSize = 5 << 20 // 5MB

var allowedImageContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var (
	client     *s3.Client
	clientOnce sync.Once
	bucket     string
	baseURL    string
)

func getClient() *s3.Client {
	clientOnce.Do(func() {
		region := config.GetEnv("AWS_REGION")
		bucket = config.GetEnv("AWS_S3_BUCKET")
		baseURL = strings.TrimRight(config.GetEnv("AWS_S3_BASE_URL"), "/")
		if baseURL == "" {
			baseURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)
		}

		accessKey := config.GetEnv("AWS_ACCESS_KEY_ID")
		secretKey := config.GetEnv("AWS_SECRET_ACCESS_KEY")

		cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
			),
		)
		if err != nil {
			panic("storage: failed to load aws config: " + err.Error())
		}

		client = s3.NewFromConfig(cfg)
	})

	return client
}

// UploadImage validates a multipart image upload and stores it in S3 under
// "<keyPrefix>/<uuid><ext>", returning its public URL.
func UploadImage(ctx context.Context, fileHeader *multipart.FileHeader, keyPrefix string) (string, error) {
	if fileHeader.Size > MaxImageSize {
		return "", fmt.Errorf("image exceeds max size of %d bytes", MaxImageSize)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	ext, ok := allowedImageContentTypes[contentType]
	if !ok {
		return "", fmt.Errorf("unsupported image type: %s", contentType)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open upload: %w", err)
	}
	defer file.Close()

	key := fmt.Sprintf("%s/%s%s", strings.Trim(keyPrefix, "/"), uuid.NewString(), ext)

	_, err = getClient().PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	return fmt.Sprintf("%s/%s", baseURL, key), nil
}
