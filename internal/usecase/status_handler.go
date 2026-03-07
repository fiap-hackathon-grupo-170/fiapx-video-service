package usecase

import (
	"context"
	"fmt"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/fiapx/fiapx-video-service/internal/infra/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type HandleStatusUpdateUseCase struct {
	repo   port.VideoRepository
	cache  port.VideoCache
	logger *zap.Logger
}

func NewHandleStatusUpdateUseCase(
	repo port.VideoRepository,
	cache port.VideoCache,
	logger *zap.Logger,
) *HandleStatusUpdateUseCase {
	return &HandleStatusUpdateUseCase{repo: repo, cache: cache, logger: logger}
}

func (uc *HandleStatusUpdateUseCase) Execute(ctx context.Context, msg entity.VideoStatusMessage) error {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "HandleStatusUpdateUseCase.Execute")
	defer span.End()

	span.SetAttributes(
		attribute.String("job.id", msg.JobID.String()),
		attribute.String("job.status", msg.Status),
	)

	log := uc.logger.With(
		zap.String("job_id", msg.JobID.String()),
		zap.String("status", msg.Status),
	)

	video, err := uc.repo.FindByID(ctx, msg.JobID)
	if err != nil {
		return fmt.Errorf("find video for status update: %w", err)
	}

	switch msg.Status {
	case "COMPLETED":
		video.MarkCompleted(msg.ZipKey, msg.FrameCount, msg.Duration)
	case "FAILED":
		video.MarkFailed(msg.ErrorMessage)
	default:
		log.Warn("received unknown status, ignoring", zap.String("status", msg.Status))
		return nil
	}

	if err := uc.repo.Update(ctx, video); err != nil {
		return fmt.Errorf("update video status: %w", err)
	}

	if err := uc.cache.InvalidateUserVideos(ctx, video.UserID); err != nil {
		log.Warn("failed to invalidate cache after status update", zap.Error(err))
	}

	metrics.VideosStatusUpdated.WithLabelValues(msg.Status).Inc()

	log.Info("video status updated successfully",
		zap.String("video_id", video.ID.String()),
		zap.String("user_id", video.UserID),
	)

	return nil
}
