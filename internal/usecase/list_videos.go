package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type ListVideosUseCase struct {
	repo   port.VideoRepository
	cache  port.VideoCache
	logger *zap.Logger
}

func NewListVideosUseCase(
	repo port.VideoRepository,
	cache port.VideoCache,
	logger *zap.Logger,
) *ListVideosUseCase {
	return &ListVideosUseCase{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (uc *ListVideosUseCase) Execute(ctx context.Context, userID string) ([]*entity.Video, error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "ListVideosUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", userID))

	cached, err := uc.cache.GetUserVideos(ctx, userID)
	if err == nil && cached != nil {
		return cached, nil
	}

	videos, err := uc.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find videos by user: %w", err)
	}

	if setErr := uc.cache.SetUserVideos(ctx, userID, videos, 60*time.Second); setErr != nil {
		uc.logger.Warn("failed to set user videos cache", zap.String("user_id", userID), zap.Error(setErr))
	}

	return videos, nil
}
