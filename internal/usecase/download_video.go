package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var (
	ErrVideoNotReady = errors.New("video ainda sendo processado")
	ErrVideoFailed   = errors.New("processamento do video falhou")
)

const presignedURLTTL = 15 * time.Minute

type DownloadVideoResult struct {
	URL            string
	ExpiresSeconds int
}

type DownloadVideoUseCase struct {
	repo    port.VideoRepository
	storage port.VideoStorage
	logger  *zap.Logger
}

func NewDownloadVideoUseCase(
	repo port.VideoRepository,
	storage port.VideoStorage,
	logger *zap.Logger,
) *DownloadVideoUseCase {
	return &DownloadVideoUseCase{repo: repo, storage: storage, logger: logger}
}

func (uc *DownloadVideoUseCase) Execute(ctx context.Context, id uuid.UUID, userID string) (*DownloadVideoResult, error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "DownloadVideoUseCase.Execute")
	defer span.End()

	span.SetAttributes(
		attribute.String("video.id", id.String()),
		attribute.String("user.id", userID),
	)

	video, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrVideoNotFound, id)
	}

	if video.UserID != userID {
		return nil, fmt.Errorf("%w: %s", ErrVideoNotFound, id)
	}

	switch video.Status {
	case entity.VideoStatusPending, entity.VideoStatusProcessing:
		return nil, ErrVideoNotReady
	case entity.VideoStatusFailed:
		return nil, ErrVideoFailed
	}

	url, err := uc.storage.PresignedDownloadURL(ctx, video.ZipKey, presignedURLTTL)
	if err != nil {
		return nil, fmt.Errorf("generate presigned url: %w", err)
	}

	return &DownloadVideoResult{
		URL:            url,
		ExpiresSeconds: int(presignedURLTTL.Seconds()),
	}, nil
}
