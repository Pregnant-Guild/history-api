package storage

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"

	ffconfig "history-api/pkg/config"
)

type UploadOptions struct {
	ContentType        string
	ContentDisposition string
	Metadata           map[string]string
}

type Storage interface {
	Upload(ctx context.Context, key string, body io.Reader, size int64, opts UploadOptions) error
	PresignUpload(ctx context.Context, key string, expire time.Duration, opts UploadOptions) (string, error)
	GetURL(ctx context.Context, key string, expire time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

type s3Storage struct {
	client   *s3.Client
	ps       *s3.PresignClient
	bucket   string
	endPoint string
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

	endpoint, err := ffconfig.GetConfig("STORAGE_ENDPOINT")
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Error().Msgf("unable to load AWS SDK config, %v", err)
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &s3Storage{
		client:   client,
		ps:       s3.NewPresignClient(client),
		bucket:   bucketName,
		endPoint: endpoint,
	}, nil
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
		Bucket: &s.bucket,
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
