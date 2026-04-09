package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	ffconfig "history-api/pkg/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"
)

type UploadOptions struct {
	ContentType        string
	ContentDisposition string
	Metadata           map[string]string
}

type MoveOptions struct {
	Bucket string
	Key    string
}

type Storage interface {
	Upload(ctx context.Context, key string, body io.Reader, size int64, opts UploadOptions) error
	Move(ctx context.Context, src *MoveOptions, dest *MoveOptions) error
	PresignUpload(ctx context.Context, key string, expire time.Duration, opts UploadOptions) (string, error)
	GetURL(ctx context.Context, key string, expire time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
	BulkDelete(ctx context.Context, keys []string) error
	GetMainBucket() string
	GetTempBucket() string
}

type s3Storage struct {
	client     *s3.Client
	ps         *s3.PresignClient
	bucket     string
	tempBucket string
	endPoint   string
}

func NewS3Storage() (Storage, error) {
	accessKey, err := ffconfig.GetConfig("STORAGE_ACCESS_KEY")
	if err != nil {
		return nil, err
	}

	secretAccessKey, err := ffconfig.GetConfig("STORAGE_SECRET_KEY")
	if err != nil {
		return nil, err
	}

	bucketName, err := ffconfig.GetConfig("STORAGE_BUCKET_NAME")
	if err != nil {
		return nil, err
	}
	tempBucketName, err := ffconfig.GetConfig("STORAGE_BUCKET_TEMP_NAME")
	if err != nil {
		return nil, err
	}

	endpoint, err := ffconfig.GetConfig("STORAGE_ENDPOINT")
	if err != nil {
		return nil, err
	}

	region, err := ffconfig.GetConfig("STORAGE_REGION")
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretAccessKey, "")),
		config.WithRegion(region),
	)

	if err != nil {
		log.Error().Msgf("unable to load AWS SDK config, %v", err)
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	return &s3Storage{
		client:     client,
		ps:         s3.NewPresignClient(client),
		bucket:     bucketName,
		tempBucket: tempBucketName,
		endPoint:   endpoint,
	}, nil
}
func (s *s3Storage) GetMainBucket() string { return s.bucket }
func (s *s3Storage) GetTempBucket() string { return s.tempBucket }

func (s *s3Storage) Move(ctx context.Context, src *MoveOptions, dest *MoveOptions) error {
	copySource := fmt.Sprintf("%s/%s", src.Bucket, url.PathEscape(src.Key))

	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(dest.Bucket),
		Key:        aws.String(dest.Key),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	waiter := s3.NewObjectExistsWaiter(s.client)
	err = waiter.Wait(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(dest.Bucket),
		Key:    aws.String(dest.Key),
	}, time.Second*10)
	if err != nil {
		return fmt.Errorf("object not available after copy: %w", err)
	}

	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(src.Bucket),
		Key:    aws.String(src.Key),
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to delete source object after copy")
	}

	return nil
}

func (s *s3Storage) Upload(ctx context.Context, key string, body io.Reader, size int64, opts UploadOptions) error {
	input := &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   body,
	}

	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	if opts.ContentDisposition != "" {
		input.ContentDisposition = aws.String(opts.ContentDisposition)
	}
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	_, err := s.client.PutObject(ctx, input)
	return err
}

func (s *s3Storage) PresignUpload(ctx context.Context, key string, expire time.Duration, opts UploadOptions) (string, error) {
	input := &s3.PutObjectInput{
		Bucket: &s.tempBucket,
		Key:    &key,
	}

	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	if opts.ContentDisposition != "" {
		input.ContentDisposition = aws.String(opts.ContentDisposition)
	}
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	req, err := s.ps.PresignPutObject(ctx, input, s3.WithPresignExpires(expire))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (s *s3Storage) GetURL(ctx context.Context, key string, expire time.Duration) (string, error) {
	req, err := s.ps.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expire))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (s *s3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	return err
}

func (s *s3Storage) BulkDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	batchSize := 1000
	var hasError bool

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		var objects []types.ObjectIdentifier
		for _, k := range batch {
			objects = append(objects, types.ObjectIdentifier{Key: aws.String(k)})
		}

		_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{Objects: objects},
		})

		if err != nil {
			log.Error().Err(err).Int("start", i).Int("end", end).Msg("S3 batch delete failed")
			hasError = true
			continue
		}
	}

	if hasError {
		return fmt.Errorf("one or more batches failed to delete")
	}
	return nil
}
