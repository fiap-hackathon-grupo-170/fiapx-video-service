package port

import (
	"context"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/google/uuid"
)

type VideoRepository interface {
	Create(ctx context.Context, video *entity.Video) error
	Update(ctx context.Context, video *entity.Video) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Video, error)
	FindByUserID(ctx context.Context, userID string) ([]*entity.Video, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
