package port

import (
	"context"
	"io"
	"time"
)

type VideoStorage interface {
	UploadVideo(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	DeleteVideo(ctx context.Context, key string) error
	PresignedDownloadURL(ctx context.Context, zipKey string, ttl time.Duration) (string, error)
}
