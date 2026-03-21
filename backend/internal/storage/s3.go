package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"comp-video-service/backend/internal/config"
)

// S3Client wraps the AWS SDK S3 client with service-specific helpers.
type S3Client struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

// NewS3Client creates and validates a new S3Client using the given config.
func NewS3Client(ctx context.Context, cfg *config.Config) (*S3Client, error) {
	//nolint:staticcheck // MinIO endpoint routing currently relies on this SDK resolver path.
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.S3Endpoint,
				HostnameImmutable: true,
			}, nil
		},
	)

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.S3Region),
		//nolint:staticcheck // See resolver note above.
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3AccessKeyID,
			cfg.S3SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.S3UsePathStyle
	})

	return &S3Client{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        cfg.S3Bucket,
	}, nil
}

// Upload streams the content to S3 under the given key with the provided content-type.
func (s *S3Client) Upload(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return fmt.Errorf("s3 put object %q: %w", key, err)
	}
	return nil
}

// PresignedURL returns a presigned GET URL for the given key, valid for the specified duration.
func (s *S3Client) PresignedURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(duration))
	if err != nil {
		return "", fmt.Errorf("presign get object %q: %w", key, err)
	}
	return req.URL, nil
}

// Delete removes an object from S3.
func (s *S3Client) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete object %q: %w", key, err)
	}
	return nil
}
