package usecase

import (
	"context"
	"fmt"

	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type DeleteVideoUseCase struct {
	repo    port.VideoRepository
	storage port.VideoStorage
	cache   port.VideoCache
	logger  *zap.Logger
}

func NewDeleteVideoUseCase(
	repo port.VideoRepository,
	storage port.VideoStorage,
	cache port.VideoCache,
	logger *zap.Logger,
) *DeleteVideoUseCase {
	return &DeleteVideoUseCase{repo: repo, storage: storage, cache: cache, logger: logger}
}

func (uc *DeleteVideoUseCase) Execute(ctx context.Context, id uuid.UUID, userID string) error {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "DeleteVideoUseCase.Execute")
	defer span.End()

	span.SetAttributes(
		attribute.String("video.id", id.String()),
		attribute.String("user.id", userID),
	)

	video, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrVideoNotFound, id)
	}

	if video.UserID != userID {
		return fmt.Errorf("%w: %s", ErrVideoNotFound, id)
	}

	if err := uc.storage.DeleteVideo(ctx, video.VideoKey); err != nil {
		uc.logger.Warn("failed to delete video from storage (ignoring)", zap.Error(err), zap.String("key", video.VideoKey))
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete video record: %w", err)
	}

	if err := uc.cache.InvalidateUserVideos(ctx, userID); err != nil {
		uc.logger.Warn("failed to invalidate cache after delete", zap.Error(err))
	}

	return nil
}
