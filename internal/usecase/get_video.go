package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var ErrVideoNotFound = errors.New("video not found")

type GetVideoUseCase struct {
	repo   port.VideoRepository
	logger *zap.Logger
}

func NewGetVideoUseCase(repo port.VideoRepository, logger *zap.Logger) *GetVideoUseCase {
	return &GetVideoUseCase{repo: repo, logger: logger}
}

func (uc *GetVideoUseCase) Execute(ctx context.Context, id uuid.UUID, userID string) (*entity.Video, error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "GetVideoUseCase.Execute")
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

	return video, nil
}
