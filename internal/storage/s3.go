package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/shridarpatil/whatomate/internal/config"
)

// S3Client provides upload and presigned URL operations for call recordings.
type S3Client struct {
	client                   *s3.Client
	bucket                   string
	serverSideEncryption     string
	kmsKeyID                 string
}

// NewS3Client creates a new S3 client from the application's StorageConfig.
func NewS3Client(cfg *config.StorageConfig) (*S3Client, error) {
	if cfg.S3Bucket == "" || cfg.S3Region == "" {
		return nil, fmt.Errorf("s3_bucket and s3_region are required")
	}

	encryption, err := normalizeServerSideEncryption(cfg.S3ServerSideEncryption, cfg.S3KMSKeyID)
	if err != nil {
		return nil, err
	}

	opts := s3.Options{
		Region: cfg.S3Region,
	}
	if cfg.S3Endpoint != "" {
		opts.BaseEndpoint = aws.String(cfg.S3Endpoint)
		opts.UsePathStyle = cfg.S3ForcePathStyle
	}

	if cfg.S3Key != "" && cfg.S3Secret != "" {
		opts.Credentials = credentials.NewStaticCredentialsProvider(cfg.S3Key, cfg.S3Secret, "")
	}

	client := s3.New(opts)
	return &S3Client{
		client:               client,
		bucket:               cfg.S3Bucket,
		serverSideEncryption: encryption,
		kmsKeyID:             cfg.S3KMSKeyID,
	}, nil
}

func normalizeServerSideEncryption(value, kmsKeyID string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		if strings.TrimSpace(kmsKeyID) != "" {
			return "", fmt.Errorf("s3_kms_key_id requires s3_server_side_encryption = aws:kms")
		}
		return "", nil
	case "aes256":
		if strings.TrimSpace(kmsKeyID) != "" {
			return "", fmt.Errorf("s3_kms_key_id can only be used with s3_server_side_encryption = aws:kms")
		}
		return "AES256", nil
	case "aws:kms":
		return "aws:kms", nil
	default:
		return "", fmt.Errorf("unsupported s3_server_side_encryption %q: use AES256 or aws:kms", value)
	}
}

// Download returns an object body from the configured bucket. The caller must
// close the returned reader.
func (s *S3Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

// Upload uploads a file to S3 at the given key.
func (s *S3Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	_, err := s.client.PutObject(ctx, s.putObjectInput(key, body, contentType))
	return err
}

func (s *S3Client) putObjectInput(key string, body io.Reader, contentType string) *s3.PutObjectInput {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if s.serverSideEncryption != "" {
		input.ServerSideEncryption = types.ServerSideEncryption(s.serverSideEncryption)
	}
	if s.kmsKeyID != "" {
		input.SSEKMSKeyId = aws.String(s.kmsKeyID)
	}
	return input
}

// Delete removes one object from the configured bucket.
func (s *S3Client) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// Archive copies an object to a new key, then removes the original. Upload
// settings, including server-side encryption, are applied to the archive copy.
func (s *S3Client) Archive(ctx context.Context, sourceKey, archiveKey string) error {
	body, err := s.Download(ctx, sourceKey)
	if err != nil {
		return err
	}
	defer body.Close()
	if err := s.Upload(ctx, archiveKey, body, "audio/ogg"); err != nil {
		return err
	}
	return s.Delete(ctx, sourceKey)
}

// GetPresignedURL returns a time-limited download URL for the given S3 key.
func (s *S3Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presigner := s3.NewPresignClient(s.client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}
