package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client       *miniogo.Client
	uploadBucket string
	zipBucket    string
	publicURL    string
}

type StorageConfig struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	UseSSL       bool
	UploadBucket string
	ZipBucket    string
	PublicURL    string
}

func NewStorage(cfg StorageConfig) (*Storage, error) {
	client, err := miniogo.New(cfg.Endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &Storage{
		client:       client,
		uploadBucket: cfg.UploadBucket,
		zipBucket:    cfg.ZipBucket,
		publicURL:    strings.TrimRight(cfg.PublicURL, "/"),
	}, nil
}

func (s *Storage) EnsureBuckets(ctx context.Context) error {
	for _, bucket := range []string{s.uploadBucket, s.zipBucket} {
		exists, err := s.client.BucketExists(ctx, bucket)
		if err != nil {
			return fmt.Errorf("check bucket %s: %w", bucket, err)
		}
		if !exists {
			if err := s.client.MakeBucket(ctx, bucket, miniogo.MakeBucketOptions{}); err != nil {
				return fmt.Errorf("create bucket %s: %w", bucket, err)
			}
		}
	}
	return nil
}

func (s *Storage) UploadVideo(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.uploadBucket, key, reader, size, miniogo.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("upload video to minio: %w", err)
	}
	return nil
}

func (s *Storage) DeleteVideo(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.uploadBucket, key, miniogo.RemoveObjectOptions{})
	if err != nil {
		errResp := miniogo.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return nil
		}
		return fmt.Errorf("delete video from minio: %w", err)
	}
	return nil
}

func (s *Storage) PresignedDownloadURL(ctx context.Context, zipKey string, ttl time.Duration) (string, error) {
	presignedURL, err := s.client.PresignedGetObject(ctx, s.zipBucket, zipKey, ttl, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign download url: %w", err)
	}

	result := presignedURL.String()

	// Reescreve o host para URL publica quando configurado (ex: minio:9000 -> localhost:9000)
	if s.publicURL != "" {
		internalOrigin := presignedURL.Scheme + "://" + presignedURL.Host
		result = strings.Replace(result, internalOrigin, s.publicURL, 1)
	}

	return result, nil
}
