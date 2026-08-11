package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Storage S3/MinIO 对象存储实现
// MinIO 使用 S3 兼容协议，同一套代码覆盖 S3 与 MinIO（通过 PathStyle + Endpoint 区分）
type S3Storage struct {
	client    *s3.Client
	bucket    string
	baseURL   string
	pathStyle bool
}

// newS3Storage 创建 S3 存储（MinIO 复用，S3 兼容协议）
func newS3Storage(cfg Config) (Storage, error) {
	return newS3CompatibleStorage(cfg, true)
}

// newMinioStorage 创建 MinIO 存储
func newMinioStorage(cfg Config) (Storage, error) {
	return newS3CompatibleStorage(cfg, false)
}

func newS3CompatibleStorage(cfg Config, isS3 bool) (Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("storage bucket is required")
	}
	optFns := []func(*awsconfig.LoadOptions) error{}
	if cfg.Endpoint != "" {
		optFns = append(optFns, awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: cfg.Endpoint, SigningRegion: cfg.Region}, nil
			}),
		))
	}
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}
	if cfg.Region != "" {
		optFns = append(optFns, awsconfig.WithRegion(cfg.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// MinIO 需路径风格；S3 默认虚拟主机风格
		if !isS3 || cfg.PathStyle {
			o.UsePathStyle = true
		}
	})
	return &S3Storage{
		client:  client,
		bucket:  cfg.Bucket,
		baseURL: strings.TrimSuffix(cfg.BaseURL, "/"),
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	return err
}

func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3Storage) Size(ctx context.Context, key string) (int64, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, err
	}
	if out.ContentLength == nil {
		return 0, nil
	}
	return *out.ContentLength, nil
}

func (s *S3Storage) URL(key string) string {
	if s.baseURL != "" {
		return s.baseURL + "/" + key
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, "", key)
}

func (s *S3Storage) PresignedURL(key string, expiry time.Duration) (string, error) {
	client := s3.NewPresignClient(s.client)
	out, err := client.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(o *s3.PresignOptions) {
		o.Expires = expiry
	})
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func (s *S3Storage) PresignedUploadURL(key string, expiry time.Duration, contentType string) (string, error) {
	client := s3.NewPresignClient(s.client)
	out, err := client.PresignPutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(o *s3.PresignOptions) {
		o.Expires = expiry
	})
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

var _ Storage = (*S3Storage)(nil)
