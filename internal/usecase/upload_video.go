package usecase

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/fiapx/fiapx-video-service/internal/infra/metrics"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type UploadVideoUseCase struct {
	repo      port.VideoRepository
	storage   port.VideoStorage
	publisher port.EventPublisher
	cache     port.VideoCache
	logger    *zap.Logger
}

func NewUploadVideoUseCase(
	repo port.VideoRepository,
	storage port.VideoStorage,
	publisher port.EventPublisher,
	cache port.VideoCache,
	logger *zap.Logger,
) *UploadVideoUseCase {
	return &UploadVideoUseCase{
		repo:      repo,
		storage:   storage,
		publisher: publisher,
		cache:     cache,
		logger:    logger,
	}
}

func (uc *UploadVideoUseCase) Execute(
	ctx context.Context,
	reader io.Reader,
	originalName string,
	fileSize int64,
	userID, userEmail string,
) (*entity.Video, error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "UploadVideoUseCase.Execute")
	defer span.End()

	start := time.Now()
	metrics.ActiveUploads.Inc()
	defer metrics.ActiveUploads.Dec()

	videoID := uuid.New()
	ext := strings.ToLower(filepath.Ext(originalName))
	videoKey := fmt.Sprintf("%s/%s%s", userID, videoID.String(), ext)

	span.SetAttributes(
		attribute.String("video.id", videoID.String()),
		attribute.String("video.key", videoKey),
		attribute.String("user.id", userID),
	)

	log := uc.logger.With(
		zap.String("video_id", videoID.String()),
		zap.String("user_id", userID),
		zap.String("video_key", videoKey),
	)

	contentType := videoContentType(ext)

	if err := uc.storage.UploadVideo(ctx, videoKey, reader, fileSize, contentType); err != nil {
		log.Error("failed to upload video to minio", zap.Error(err))
		return nil, fmt.Errorf("upload video: %w", err)
	}

	video := entity.NewVideo(userID, userEmail, videoKey, originalName, fileSize)
	video.ID = videoID

	if err := uc.repo.Create(ctx, video); err != nil {
		log.Error("failed to create video record", zap.Error(err))
		return nil, fmt.Errorf("create video: %w", err)
	}

	msg := entity.VideoUploadedMessage{
		JobID:     video.ID,
		UserID:    userID,
		UserEmail: userEmail,
		VideoKey:  videoKey,
		FileSize:  fileSize,
	}
	if err := uc.publisher.PublishVideoUploaded(ctx, msg); err != nil {
		log.Error("failed to publish video uploaded event", zap.Error(err))
		return nil, fmt.Errorf("publish event: %w", err)
	}

	if err := uc.cache.InvalidateUserVideos(ctx, userID); err != nil {
		log.Warn("failed to invalidate cache", zap.Error(err))
	}

	metrics.VideosUploadedTotal.Inc()
	metrics.UploadDuration.Observe(time.Since(start).Seconds())

	log.Info("video uploaded successfully",
		zap.String("original_name", originalName),
		zap.Int64("file_size", fileSize),
	)

	return video, nil
}

func videoContentType(ext string) string {
	switch ext {
	case ".webm":
		return "video/webm"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".mkv":
		return "video/x-matroska"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	default:
		return "video/mp4"
	}
}
