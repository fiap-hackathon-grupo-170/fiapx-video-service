package port

import (
	"context"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
)

type EventPublisher interface {
	PublishVideoUploaded(ctx context.Context, msg entity.VideoUploadedMessage) error
}

type StatusConsumer interface {
	Start(ctx context.Context) error
	Close()
}
