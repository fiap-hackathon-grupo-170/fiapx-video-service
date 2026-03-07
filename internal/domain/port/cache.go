package port

import (
	"context"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
)

type VideoCache interface {
	GetUserVideos(ctx context.Context, userID string) ([]*entity.Video, error)
	SetUserVideos(ctx context.Context, userID string, videos []*entity.Video, ttl time.Duration) error
	InvalidateUserVideos(ctx context.Context, userID string) error
}
